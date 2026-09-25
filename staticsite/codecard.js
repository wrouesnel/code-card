/*
 * Code card generator: a browser port of pkg/codecard. Encoded tables are identical to those
 * produced by the Go and original Python implementations for the same salt, code and keyword.
 *
 * This is a classic script (not a module) so the page also works when opened from file://.
 * It exposes everything on globalThis.CodeCard, and has no dependencies on browser APIs other
 * than crypto.getRandomValues for random tables, so it can be tested outside a browser.
 */
(function (root) {
  "use strict";

  const LETTERS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";
  const DIGITS = "0123456789";
  // Every digit is assigned to a letter this many times before the remaining letters are filled
  // randomly, to ensure diversity and balance repeats.
  const FULL_SAMPLES = Math.floor(LETTERS.length / DIGITS.length);

  const MIN_SALT_LENGTH = 16;
  const MAX_REPEATS = 3;
  const DEFAULT_SALT = "1337code8Ackz0rs";

  // SHA-2 round constants and initial hash values.
  const K256 = [
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
    ];
  const H256 = [
    0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
    ];
  const K512 = [
    0x428a2f98d728ae22n, 0x7137449123ef65cdn, 0xb5c0fbcfec4d3b2fn, 0xe9b5dba58189dbbcn,
    0x3956c25bf348b538n, 0x59f111f1b605d019n, 0x923f82a4af194f9bn, 0xab1c5ed5da6d8118n,
    0xd807aa98a3030242n, 0x12835b0145706fben, 0x243185be4ee4b28cn, 0x550c7dc3d5ffb4e2n,
    0x72be5d74f27b896fn, 0x80deb1fe3b1696b1n, 0x9bdc06a725c71235n, 0xc19bf174cf692694n,
    0xe49b69c19ef14ad2n, 0xefbe4786384f25e3n, 0x0fc19dc68b8cd5b5n, 0x240ca1cc77ac9c65n,
    0x2de92c6f592b0275n, 0x4a7484aa6ea6e483n, 0x5cb0a9dcbd41fbd4n, 0x76f988da831153b5n,
    0x983e5152ee66dfabn, 0xa831c66d2db43210n, 0xb00327c898fb213fn, 0xbf597fc7beef0ee4n,
    0xc6e00bf33da88fc2n, 0xd5a79147930aa725n, 0x06ca6351e003826fn, 0x142929670a0e6e70n,
    0x27b70a8546d22ffcn, 0x2e1b21385c26c926n, 0x4d2c6dfc5ac42aedn, 0x53380d139d95b3dfn,
    0x650a73548baf63den, 0x766a0abb3c77b2a8n, 0x81c2c92e47edaee6n, 0x92722c851482353bn,
    0xa2bfe8a14cf10364n, 0xa81a664bbc423001n, 0xc24b8b70d0f89791n, 0xc76c51a30654be30n,
    0xd192e819d6ef5218n, 0xd69906245565a910n, 0xf40e35855771202an, 0x106aa07032bbd1b8n,
    0x19a4c116b8d2d0c8n, 0x1e376c085141ab53n, 0x2748774cdf8eeb99n, 0x34b0bcb5e19b48a8n,
    0x391c0cb3c5c95a63n, 0x4ed8aa4ae3418acbn, 0x5b9cca4f7763e373n, 0x682e6ff3d6b2b8a3n,
    0x748f82ee5defb2fcn, 0x78a5636f43172f60n, 0x84c87814a1f0ab72n, 0x8cc702081a6439ecn,
    0x90befffa23631e28n, 0xa4506cebde82bde9n, 0xbef9a3f7b2c67915n, 0xc67178f2e372532bn,
    0xca273eceea26619cn, 0xd186b8c721c0c207n, 0xeada7dd6cde0eb1en, 0xf57d4f7fee6ed178n,
    0x06f067aa72176fban, 0x0a637dc5a2c898a6n, 0x113f9804bef90daen, 0x1b710b35131c471bn,
    0x28db77f523047d84n, 0x32caab7b40c72493n, 0x3c9ebe0a15c9bebcn, 0x431d67c49c100d4cn,
    0x4cc5d4becb3e42b6n, 0x597f299cfc657e2an, 0x5fcb6fab3ad6faecn, 0x6c44198c4a475817n,
    ];
  const H512 = [
    0x6a09e667f3bcc908n, 0xbb67ae8584caa73bn, 0x3c6ef372fe94f82bn, 0xa54ff53a5f1d36f1n,
    0x510e527fade682d1n, 0x9b05688c2b3e6c1fn, 0x1f83d9abfb41bd6bn, 0x5be0cd19137e2179n,
    ];

  // utf8 encodes str as an array of UTF-8 bytes.
  function utf8(str) {
    const encoded = encodeURIComponent(str);
    const bytes = [];
    for (let idx = 0; idx < encoded.length; idx++) {
      if (encoded[idx] === "%") {
        bytes.push(parseInt(encoded.substr(idx + 1, 2), 16));
        idx += 2;
      } else {
        bytes.push(encoded.charCodeAt(idx));
      }
    }
    return bytes;
  }

  // shaPad applies SHA-2 message padding for the given block and length field sizes.
  function shaPad(message, blockSize, lengthSize) {
    const padded = message.concat([0x80]);
    while (padded.length % blockSize !== blockSize - lengthSize) {
      padded.push(0);
    }
    let bitLength = message.length * 8;
    const lengthBytes = [];
    for (let idx = 0; idx < lengthSize; idx++) {
      lengthBytes.unshift(bitLength % 256);
      bitLength = Math.floor(bitLength / 256);
    }
    return padded.concat(lengthBytes);
  }

  function rotr32(x, n) {
    return ((x >>> n) | (x << (32 - n))) >>> 0;
  }

  // sha256 returns the SHA-256 digest of a byte array.
  function sha256(message) {
    const h = H256.slice();
    const data = shaPad(message, 64, 8);
    const w = new Array(64);
    for (let offset = 0; offset < data.length; offset += 64) {
      for (let i = 0; i < 16; i++) {
        const b = offset + i * 4;
        w[i] = ((data[b] << 24) | (data[b + 1] << 16) | (data[b + 2] << 8) | data[b + 3]) >>> 0;
      }
      for (let i = 16; i < 64; i++) {
        const s0 = rotr32(w[i - 15], 7) ^ rotr32(w[i - 15], 18) ^ (w[i - 15] >>> 3);
        const s1 = rotr32(w[i - 2], 17) ^ rotr32(w[i - 2], 19) ^ (w[i - 2] >>> 10);
        w[i] = (w[i - 16] + s0 + w[i - 7] + s1) >>> 0;
      }
      let [a, b, c, d, e, f, g, hh] = h;
      for (let i = 0; i < 64; i++) {
        const S1 = rotr32(e, 6) ^ rotr32(e, 11) ^ rotr32(e, 25);
        const ch = (e & f) ^ (~e & g);
        const t1 = (hh + S1 + ch + K256[i] + w[i]) >>> 0;
        const S0 = rotr32(a, 2) ^ rotr32(a, 13) ^ rotr32(a, 22);
        const maj = (a & b) ^ (a & c) ^ (b & c);
        const t2 = (S0 + maj) >>> 0;
        hh = g;
        g = f;
        f = e;
        e = (d + t1) >>> 0;
        d = c;
        c = b;
        b = a;
        a = (t1 + t2) >>> 0;
      }
      [a, b, c, d, e, f, g, hh].forEach((value, idx) => {
        h[idx] = (h[idx] + value) >>> 0;
      });
    }
    const digest = [];
    for (const word of h) {
      digest.push(word >>> 24, (word >>> 16) & 0xff, (word >>> 8) & 0xff, word & 0xff);
    }
    return digest;
  }

  const MASK64 = (1n << 64n) - 1n;

  function rotr64(x, n) {
    return ((x >> n) | (x << (64n - n))) & MASK64;
  }

  // sha512 returns the SHA-512 digest of a byte array.
  function sha512(message) {
    const h = H512.slice();
    const data = shaPad(message, 128, 16);
    const w = new Array(80);
    for (let offset = 0; offset < data.length; offset += 128) {
      for (let i = 0; i < 16; i++) {
        let word = 0n;
        for (let b = 0; b < 8; b++) {
          word = (word << 8n) | BigInt(data[offset + i * 8 + b]);
        }
        w[i] = word;
      }
      for (let i = 16; i < 80; i++) {
        const s0 = rotr64(w[i - 15], 1n) ^ rotr64(w[i - 15], 8n) ^ (w[i - 15] >> 7n);
        const s1 = rotr64(w[i - 2], 19n) ^ rotr64(w[i - 2], 61n) ^ (w[i - 2] >> 6n);
        w[i] = (w[i - 16] + s0 + w[i - 7] + s1) & MASK64;
      }
      let [a, b, c, d, e, f, g, hh] = h;
      for (let i = 0; i < 80; i++) {
        const S1 = rotr64(e, 14n) ^ rotr64(e, 18n) ^ rotr64(e, 41n);
        const ch = (e & f) ^ (~e & MASK64 & g);
        const t1 = (hh + S1 + ch + K512[i] + w[i]) & MASK64;
        const S0 = rotr64(a, 28n) ^ rotr64(a, 34n) ^ rotr64(a, 39n);
        const maj = (a & b) ^ (a & c) ^ (b & c);
        const t2 = (S0 + maj) & MASK64;
        hh = g;
        g = f;
        f = e;
        e = (d + t1) & MASK64;
        d = c;
        c = b;
        b = a;
        a = (t1 + t2) & MASK64;
      }
      [a, b, c, d, e, f, g, hh].forEach((value, idx) => {
        h[idx] = (h[idx] + value) & MASK64;
      });
    }
    const digest = [];
    for (const word of h) {
      for (let b = 7; b >= 0; b--) {
        digest.push(Number((word >> BigInt(b * 8)) & 0xffn));
      }
    }
    return digest;
  }

  const MT_N = 624;
  const MT_M = 397;

  // PyRandom is a Mersenne Twister (MT19937) seeded and consumed exactly as CPython's
  // random.Random(bytes) is. See pkg/pyrandom. It is NOT cryptographically secure.
  class PyRandom {
    // seed is a byte array. Like CPython's seeding version 2, it is extended with its SHA-512
    // digest and interpreted as a big-endian integer, which is split into 32-bit words least
    // significant first.
    constructor(seed) {
      const material = seed.concat(sha512(seed)).reverse();
      while (material.length % 4 !== 0) {
        material.push(0);
      }
      const key = [];
      for (let idx = 0; idx < material.length; idx += 4) {
        key.push((material[idx] | (material[idx + 1] << 8) | (material[idx + 2] << 16) | (material[idx + 3] << 24)) >>> 0);
      }
      // Only as many words as the integer's bit length requires are used (minimum 1).
      while (key.length > 1 && key[key.length - 1] === 0) {
        key.pop();
      }
      this.state = new Uint32Array(MT_N);
      this.index = MT_N;
      this.initByArray(key);
    }

    initGenrand(seed) {
      const mt = this.state;
      mt[0] = seed;
      for (let i = 1; i < MT_N; i++) {
        const prev = mt[i - 1];
        mt[i] = Math.imul(1812433253, prev ^ (prev >>> 30)) + i;
      }
      this.index = MT_N;
    }

    initByArray(key) {
      const mt = this.state;
      this.initGenrand(19650218);
      let i = 1;
      let j = 0;
      for (let k = Math.max(MT_N, key.length); k > 0; k--) {
        const prev = mt[i - 1];
        mt[i] = ((mt[i] ^ Math.imul(prev ^ (prev >>> 30), 1664525)) >>> 0) + key[j] + j;
        i++;
        j++;
        if (i >= MT_N) {
          mt[0] = mt[MT_N - 1];
          i = 1;
        }
        if (j >= key.length) {
          j = 0;
        }
      }
      for (let k = MT_N - 1; k > 0; k--) {
        const prev = mt[i - 1];
        mt[i] = ((mt[i] ^ Math.imul(prev ^ (prev >>> 30), 1566083941)) >>> 0) - i;
        i++;
        if (i >= MT_N) {
          mt[0] = mt[MT_N - 1];
          i = 1;
        }
      }
      mt[0] = 0x80000000;
    }

    // uint32 returns the next 32-bit output (CPython's genrand_uint32).
    uint32() {
      const mt = this.state;
      if (this.index >= MT_N) {
        for (let k = 0; k < MT_N; k++) {
          const y = (mt[k] & 0x80000000) | (mt[(k + 1) % MT_N] & 0x7fffffff);
          mt[k] = mt[(k + MT_M) % MT_N] ^ (y >>> 1) ^ (y & 1 ? 0x9908b0df : 0);
        }
        this.index = 0;
      }
      let y = mt[this.index++];
      y ^= y >>> 11;
      y ^= (y << 7) & 0x9d2c5680;
      y ^= (y << 15) & 0xefc60000;
      y ^= y >>> 18;
      return y >>> 0;
    }

    // getRandBits is CPython's getrandbits(k) for 0 <= k <= 32.
    getRandBits(k) {
      if (k <= 0) {
        return 0;
      }
      return this.uint32() >>> (32 - k);
    }

    // randBelow returns a value in [0, n) as CPython's _randbelow_with_getrandbits does.
    randBelow(n) {
      const k = 32 - Math.clz32(n);
      for (;;) {
        const value = this.getRandBits(k);
        if (value < n) {
          return value;
        }
      }
    }
  }

  // cryptoRandBelow returns a uniform value in [0, n) from the browser's secure random source.
  function cryptoRandBelow(n) {
    const limit = Math.floor(0x100000000 / n) * n;
    const buf = new Uint32Array(1);
    for (;;) {
      root.crypto.getRandomValues(buf);
      if (buf[0] < limit) {
        return buf[0] % n;
      }
    }
  }

  // sourceTable returns the base letter to digit assignment: every digit is used FULL_SAMPLES
  // times in order, and the remaining letters are chosen with choice.
  function sourceTable(choice) {
    const src = [];
    for (let idx = 0; idx < FULL_SAMPLES * DIGITS.length; idx++) {
      src.push(DIGITS[idx % DIGITS.length]);
    }
    while (src.length < LETTERS.length) {
      src.push(choice());
    }
    return src;
  }

  const SALT_ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  const RANDOM_SALT_LENGTH = 24;

  // randomSalt returns a random alphanumeric salt of RANDOM_SALT_LENGTH characters (about 143
  // bits). randBelow defaults to the secure random source.
  function randomSalt(randBelow) {
    const rand = randBelow || cryptoRandBelow;
    let salt = "";
    for (let idx = 0; idx < RANDOM_SALT_LENGTH; idx++) {
      salt += SALT_ALPHABET[rand(SALT_ALPHABET.length)];
    }
    return salt;
  }

  // randomTable generates a totally random table. A table is an array of 26 arrays of digit
  // characters, one per letter A-Z. randBelow defaults to the secure random source.
  function randomTable(randBelow) {
    const rand = randBelow || cryptoRandBelow;
    const src = sourceTable(() => DIGITS[rand(DIGITS.length)]);
    for (let i = src.length - 1; i > 0; i--) {
      const j = rand(i + 1);
      [src[i], src[j]] = [src[j], src[i]];
    }
    return src.map((digit) => [digit]);
  }

  // semiRandomTable generates a deterministic table which encodes code against keyword: reading
  // the digits of the keyword's letters from the card yields the code. The nth use of a letter
  // in the keyword reads the nth digit of that letter. Throws an Error on invalid input.
  function semiRandomTable(salt, code, keyword) {
    const saltBytes = utf8(salt);
    const codeBytes = utf8(code);
    const keywordBytes = utf8(keyword);

    if (saltBytes.length < MIN_SALT_LENGTH) {
      throw new Error(`Please set a salt which is at least ${MIN_SALT_LENGTH} bytes long (yours: ${saltBytes.length} bytes).`);
    }
    if (keywordBytes.length < codeBytes.length) {
      throw new Error("The keyword must be at least as long as the code.");
    }
    if (!/^[0-9]*$/.test(code)) {
      throw new Error("The code may only contain the digits 0-9.");
    }
    const upperKeyword = keyword.toUpperCase();
    if (!/^[A-Z]+$/.test(upperKeyword)) {
      throw new Error("The keyword may only contain the letters A-Z.");
    }

    const counts = {};
    let repeats = 0;
    for (const letter of upperKeyword) {
      counts[letter] = (counts[letter] || 0) + 1;
      repeats = Math.max(repeats, counts[letter]);
    }
    if (repeats > MAX_REPEATS) {
      throw new Error(`Too many repeated characters in the keyword: ${repeats} > ${MAX_REPEATS}.`);
    }

    const rng = new PyRandom(sha256(saltBytes.concat(codeBytes, keywordBytes)));
    const choice = () => DIGITS[rng.randBelow(DIGITS.length)];

    // Build a table with random repeats in it, up to the maximum.
    const table = sourceTable(choice).map((digit) => {
      const values = [digit];
      for (let idx = 1; idx < repeats; idx++) {
        values.push(choice());
      }
      return values;
    });

    // Set the letters we must. Since the table was built with the maximum number of repeats,
    // every position exists.
    const setCounts = {};
    for (let idx = 0; idx < code.length; idx++) {
      const letter = upperKeyword[idx];
      const position = setCounts[letter] || 0;
      table[LETTERS.indexOf(letter)][position] = code[idx];
      setCounts[letter] = position + 1;
    }
    return table;
  }

  // tableRows returns display strings of the form "A 1 2", in letter order.
  function tableRows(table) {
    return table.map((values, idx) => [LETTERS[idx]].concat(values).join(" "));
  }

  root.CodeCard = {
    DEFAULT_SALT,
    MIN_SALT_LENGTH,
    MAX_REPEATS,
    utf8,
    sha256,
    sha512,
    PyRandom,
    randomSalt,
    randomTable,
    semiRandomTable,
    tableRows,
  };
})(globalThis);
