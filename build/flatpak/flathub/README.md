# Flathub Submission

This directory contains the assets for submitting Aerion to Flathub using **pre-built binaries** (extra-data approach) after each Github release.

## Why Pre-Built Binaries?

Aerion uses the extra-data approach (similar to Discord, Spotify) because OAuth credentials are embedded at build time in GitHub Actions. This allows users to have Gmail/Outlook OAuth working out-of-the-box without exposing secrets to Flathub's build infrastructure.

## Prerequisites

Before submitting to Flathub:

1. **Create a GitHub release** with version tag (e.g., `v0.1.14`)
2. **Ensure release includes**:
   - `aerion-v0.1.13-linux-x86_64.tar.gz` (x86_64 binary with OAuth credentials)
   - `aerion-v0.1.13-linux-aarch64.tar.gz` (aarch64 binary with OAuth credentials)

## Updating the Manifest for New Releases

Use the provided script to calculate SHA256 hashes and automatically update the manifest:

```bash
./calculate-hashes.sh v0.1.14
```

This will:
- Download the release tarballs from GitHub
- Calculate SHA256 hashes and file sizes
- Automatically update `com.github.hkdb.Aerion-extradata.yml` with new values
- Create a backup of the original file

## Initial Flathub Submission (v0.1.14 - Ready Now!)

### Step 1: Fork flathub/flathub

**Go to**: https://github.com/flathub/flathub/fork

**CRITICAL**: **Uncheck** "Copy the master branch only" - you need the `new-pr` branch!

### Step 2: Clone and Create Submission Branch

```bash
# Clone your fork starting from new-pr branch
git clone --branch=new-pr git@github.com:YOUR_USERNAME/flathub.git
cd flathub

# Create your submission branch
git checkout -b add-aerion new-pr
```

### Step 3: Copy Required Files (5 files total)

```bash
# Copy manifest (rename to standard name)
cp /path/to/aerion/build/flatpak/flathub/com.github.hkdb.Aerion-extradata.yml \
   com.github.hkdb.Aerion.yml

# Copy metainfo
cp /path/to/aerion/build/flatpak/com.github.hkdb.Aerion.metainfo.xml .

# Copy desktop file (rename to use app ID)
cp /path/to/aerion/build/linux/aerion.desktop \
   com.github.hkdb.Aerion.desktop

# Copy icon (rename to use app ID)
cp /path/to/aerion/build/appicon.png \
   com.github.hkdb.Aerion.png

# Create flathub.json
cat > flathub.json << 'EOF'
{
  "only-arches": ["x86_64", "aarch64"]
}
EOF
```

### Step 4: Commit and Push

```bash
git add .
git commit -m "Add com.github.hkdb.Aerion"
git push origin add-aerion
```

### Step 5: Create Pull Request

On GitHub, create a pull request:
- **Base repository**: `flathub/flathub`
- **Base branch**: `new-pr` ← **CRITICAL!**
- **Head repository**: `YOUR_USERNAME/flathub`
- **Compare branch**: `add-aerion`
- **Title**: `Add com.github.hkdb.Aerion`

### Step 6: Review Process

Flathub reviewers will:
- Review manifest correctness
- Check metadata completeness
- Request changes if needed

**Common feedback**:
- May ask to restrict `--filesystem=home` to more specific paths
- Verify extra-data checksums match

Comment `bot, build` to trigger a test build once reviewers are satisfied.

### Step 7: Approval & Repository Creation

After approval:
- Flathub creates `flathub/com.github.hkdb.Aerion` repository
- You receive write access invitation (accept within 1 week)
- Must have 2FA enabled on GitHub

## Updating on Flathub (For Future Releases)

After v0.1.14 is on Flathub, for subsequent releases (v0.1.15, v0.1.16, etc.):

```bash
# 1. Create GitHub release with new binaries (GitHub Actions does this)

# 2. Calculate hashes and update manifest
cd /path/to/aerion/build/flatpak/flathub
./calculate-hashes.sh v0.1.15

# 3. Commit changes to Aerion repository
git add .
git commit -m "v0.1.15 - Flathub submission"
git push

# 4. Release to Flathub repository (using release.sh helper script)
./release.sh /path/to/flathub/com.github.hkdb.Aerion
# Script automatically copies: manifest, metainfo, desktop, icon, and flathub.json

cd /path/to/flathub/com.github.hkdb.Aerion
git add .
git commit -m "Update to v0.1.15"
git push

# Flathub auto-builds and publishes (no re-review needed!)
```

## Files in This Directory

- `com.github.hkdb.Aerion-extradata.yml` - Flatpak manifest using pre-built binaries (copy to Flathub as `com.github.hkdb.Aerion.yml`)
- `com.github.hkdb.Aerion.yml` - Alternative manifest (builds from source, not used for Flathub)
- `calculate-hashes.sh` - Helper script that automatically updates the manifest with new release hashes
- `release.sh` - Helper script that copies all files to the Flathub repository
- `README.md` - This file

**Files to copy from parent directory for Flathub submission:**
- `../com.github.hkdb.Aerion.metainfo.xml` - AppStream metadata
- `../../linux/aerion.desktop` - Desktop file (rename to `com.github.hkdb.Aerion.desktop`)
- `../../appicon.png` - Application icon (rename to `com.github.hkdb.Aerion.png`)

## OAuth Credentials

With the extra-data approach, OAuth credentials are **already embedded** in the pre-built binaries. Users get working Gmail/Outlook OAuth out-of-the-box without any additional configuration.

## Resources

- [Flathub Submission Guide](https://docs.flathub.org/docs/for-app-authors/submission)
- [App Requirements](https://docs.flathub.org/docs/for-app-authors/requirements)
- [Flathub Review Guidelines](https://docs.flathub.org/docs/for-app-authors/review-guidelines)
- [Extra Data Documentation](https://docs.flatpak.org/en/latest/flatpak-builder-command-reference.html#extra-data-sources)

## Troubleshooting

**Build fails with "Could not download file"**:
- Ensure release tarballs are publicly accessible on GitHub
- Verify URLs match exactly (case-sensitive)

**SHA256 mismatch error**:
- Re-run `./calculate-hashes.sh` with correct version
- Ensure you're pointing to the correct GitHub release tag

**Permission errors during runtime**:
- Review `finish-args` in manifest
- May need to justify or restrict filesystem access
