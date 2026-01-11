#!/bin/bash
# Update all import paths to new modular structure

echo "Updating import paths..."

# Find all Go files in new locations
find pkg/core pkg/backend internal cmd/shedman -name "*.go" -type f | while read file; do
  # Update imports from pkg/shedman to new locations
  sed -i '
    s|github.com/theshedman/shedman/pkg/shedman/backend|github.com/theshedman/shedman/pkg/backend|g
    s|github.com/theshedman/shedman/pkg/shedman/config|github.com/theshedman/shedman/internal/config|g
    s|github.com/theshedman/shedman/pkg/shedman/output|github.com/theshedman/shedman/internal/output|g
    s|github.com/theshedman/shedman/pkg/shedman/cache|github.com/theshedman/shedman/internal/cache|g
    s|github.com/theshedman/shedman/pkg/shedman/http|github.com/theshedman/shedman/internal/http|g
    s|github.com/theshedman/shedman/pkg/shedman/signals|github.com/theshedman/shedman/internal/signals|g
    s|github.com/theshedman/shedman/pkg/shedman/resolver|github.com/theshedman/shedman/pkg/core/resolver|g
    s|github.com/theshedman/shedman/pkg/shedman/installer|github.com/theshedman/shedman/pkg/core/installer|g
    s|github.com/theshedman/shedman/pkg/shedman/pkgdb|github.com/theshedman/shedman/pkg/core/pkgdb|g
    s|github.com/theshedman/shedman/pkg/shedman|github.com/theshedman/shedman/pkg/core|g
  ' "$file"
done

echo "Done!"
