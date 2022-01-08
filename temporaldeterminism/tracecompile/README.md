
(under research)

Attempt code rewrites to catch non-determinism. Approach:

* Probably use `-overlay` to replace code during compile time

Currently sucks because:

* Can't really stop it, because reflection or global pointer ref followed by mutation or other things cannot be caught