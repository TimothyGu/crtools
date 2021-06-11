# crtools

Some tools written to help Chromium development.

- remove\_base\_macros: Remove unneeded `#include "base/macros.h"` lines.
- remove\_disallow: Expands and inlines various `DISALLOW_` macros. See
  [Chromium bug 1010217](https://crbug.com/1010217) for more details.
