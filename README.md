# Mapsim
Simulates a hashmap with a CSPRNG (/dev/urandom), storing random values as keys and the number of collisions as values.

The purpose of this simulator is to study the memory requirements for the Proof-of-Work function found here:

http://6857coin.csail.mit.edu:8080/

We define this Proof-of-Work function as finding k hashes that collide in the top n bits.

Four command line flags are currently supported:

diff, cols, iters, and debug.

The simulator will solve for a range of k & n, with iters determining how many times each n & k will be simulated.

The lower bounds are 1 for n and 2 for k, i.e. a 1-bit collision.

The diff flag sets the upper bound for n, and defaults to 32.

The cols flag sets the upper bound for k, and defaults to 3.

The iters flag sets the number of simulations per n & k, and defaults to 100.

The debug flag will cause the simulator to print the full state of the hashmap when the solution was found, along with other information. This flag is false by default.

To capture the output data in a file for later analysis, simply pipe the output to some file.

To Do-

-Add flags to define the bottom of the ranges for n & k.

-Properly implement writing output to a file.

-Concurrency?
