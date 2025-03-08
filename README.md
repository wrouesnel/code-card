# Code Card Generator

Utility to generate a credit-card sized grid of random letter number associations.

The idea is that if you need help to memorize PIN numbers, then you choose a secret word,
and use the numbers on the card to pick your PIN number. When you need a new PIN number,
just print a new card and set the new PIN to the numbers mapped by your secret word.

An encoding function exists for shared PIN numbers: specify both the number and secret
word to generate a randomized card which correctly encodes the PIN.