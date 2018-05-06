# Mapsim
Simulates a hashmap with a CSPRNG (/dev/urandom), storing random values as keys and the number of collisions as values.

The purpose of this simulator is to study the memory requirements for the Proof-of-Work function found here:

http://6857coin.csail.mit.edu:8080/

We define this Proof-of-Work function as finding k hashes that collide in the top n bits.

The simulator will solve for a range of k & n. The lower bounds are 1 for n and 2 for k, i.e. a 1-bit collision. The "diff" and "cols" flags set upper bounds for n & k, respectively.

Run "mapsim --help" or read the source to learn more about command line arguments.

To capture the output data in a file for later analysis, pipe the output to some file with debug off.

To Do-

* Amortize simulator runs.

* Properly implement the savings of results to disk.

* Add flags to define the bottom of the ranges for n & k.

The threaded branch has a multi-threaded version - work in progress.
