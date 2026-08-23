#!/usr/bin/env sh
# Download the latest msc GitHub Release for this OS/arch, verify checksums.txt,
# and install the binary. Override with MSC_REPO, MSC_VERSION, MSC_INSTALL_DIR,
# MSC_GITHUB_TOKEN.
set -eu

repo="${MSC_REPO:-SoheilHasankhani/msc-cli}"
install_dir="${MSC_INSTALL_DIR:-${HOME}/.local/bin}"

need() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "msc install: missing required command: $1" >&2
		exit 1
	fi
}

need curl
need tar

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$os" in
	linux) ;;
	darwin) ;;
	msys*|mingw*|cygwin*)
		echo "msc install: on Windows use scripts/install.ps1" >&2
		exit 1
		;;
	*)
		echo "msc install: unsupported OS: $os" >&2
		exit 1
		;;
esac
case "$arch" in
	x86_64|amd64) arch=amd64 ;;
	aarch64|arm64) arch=arm64 ;;
	*)
		echo "msc install: unsupported architecture: $arch" >&2
		exit 1
		;;
esac

curl_gh() {
	# $1 = URL, rest forwarded to curl (output flags).
	url=$1
	shift
	if [ -n "${MSC_GITHUB_TOKEN:-}" ]; then
		set -- -H "Authorization: Bearer ${MSC_GITHUB_TOKEN}" "$@"
	fi
	case "$url" in
		https://api.github.com/*)
			curl -fsSL -H "Accept: application/vnd.github+json" "$@" "$url"
			;;
		*)
			curl -fsSL "$@" "$url"
			;;
	esac
}

if [ -n "${MSC_VERSION:-}" ]; then
	tag="v${MSC_VERSION#v}"
else
	api="https://api.github.com/repos/${repo}/releases/latest"
	json=$(curl_gh "$api")
	tag=$(printf '%s\n' "$json" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)
	if [ -z "$tag" ]; then
		echo "msc install: could not read latest tag from ${api}" >&2
		echo "set MSC_VERSION (for example 0.1.0) or MSC_GITHUB_TOKEN if GitHub rate-limited the request" >&2
		exit 1
	fi
fi

version=${tag#v}
asset="msc_${version}_${os}_${arch}.tar.gz"
base="https://github.com/${repo}/releases/download/${tag}"

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

echo "msc install: downloading ${asset} from ${repo} ${tag}" >&2
curl_gh "${base}/${asset}" -o "${tmpdir}/${asset}"
curl_gh "${base}/checksums.txt" -o "${tmpdir}/checksums.txt"

want=$(awk -v f="$asset" '$NF == f { print $1; exit }' "${tmpdir}/checksums.txt")
if [ -z "$want" ]; then
	echo "msc install: checksums.txt has no entry for ${asset}" >&2
	exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
	got=$(sha256sum "${tmpdir}/${asset}" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
	got=$(shasum -a 256 "${tmpdir}/${asset}" | awk '{ print $1 }')
else
	echo "msc install: need sha256sum or shasum to verify the download" >&2
	exit 1
fi

if [ "$got" != "$want" ]; then
	echo "msc install: checksum mismatch for ${asset}" >&2
	echo "  want ${want}" >&2
	echo "  got  ${got}" >&2
	exit 1
fi

mkdir -p "${tmpdir}/out"
tar -xzf "${tmpdir}/${asset}" -C "${tmpdir}/out"
bin=$(find "${tmpdir}/out" -type f -name msc | head -n 1)
if [ -z "$bin" ]; then
	echo "msc install: archive did not contain msc" >&2
	exit 1
fi

mkdir -p "$install_dir"
cp "$bin" "${install_dir}/msc"
chmod 755 "${install_dir}/msc"

echo "installed ${install_dir}/msc (${tag})" >&2

MSC_INSTALL_DIR="${install_dir}" "${install_dir}/msc" path install || {
	echo "msc install: could not configure PATH automatically; add ${install_dir} to PATH" >&2
}
MSC_INSTALL_DIR="${install_dir}" "${install_dir}/msc" completion install || true
"${install_dir}/msc" --version || true
echo "register a project with: msc init --repo <git-ssh-url> --path <meta-repo>" >&2
