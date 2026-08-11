# AUR package

`PKGBUILD` builds poundai from a tagged source archive. `.SRCINFO` is committed
because the AUR requires both files.

## First publication

1. Create an AUR account and add an SSH public key to it.
2. Add the corresponding private key as the `AUR_SSH_PRIVATE_KEY` GitHub Actions
   repository secret.
3. Add the GitHub Actions repository variable `AUR_ENABLED` with the value `true`.
4. Run the Release workflow. Its `publish-aur` job creates or updates the
   `poundai` AUR Git repository after the GitHub release is published.

The SSH key should be dedicated to AUR publishing. The workflow commits as
`poundai release bot <elee1766@users.noreply.github.com>`; change those values in
`.github/workflows/release.yml` if the AUR account uses different maintainer
details.

## Local validation

On Arch Linux, run:

```sh
cd packaging/aur
makepkg --cleanbuild --syncdeps
namcap PKGBUILD poundai-*.pkg.tar.zst
```

The project does not currently declare a software license, so the package uses
`LicenseRef-Unknown`. Replace it and install the repository's license file from
`package()` once a license is chosen. Until then, the package installs a notice
that the upstream license is undeclared.
