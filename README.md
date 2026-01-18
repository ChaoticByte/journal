# Journal

A simple encrypted journal for the terminal.

## Compatibility

This software is developed and intended to be used on Linux systems.  
You may struggle on old CPUs with less than 4 cores.

## Usage

Open (or create a new) journal using

```
./journal /path/to/your/journal
```

## Security

This software uses XChacha20-Poly1305 as an authenticated encryption algorithm.  
For key derivation, Argon2id is used with sensible parameters.  

The password is secured by memguard as soon as it is read into memory.

When the program exits, the terminal is cleared.
