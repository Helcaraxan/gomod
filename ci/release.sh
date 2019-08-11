#!/usr/bin/env bash
# vim: set tabstop=4 shiftwidth=4 expandtab
set -e -u -o pipefail

PROJECT_ROOT="$(dirname "${BASH_SOURCE[0]}")/.."
cd "${PROJECT_ROOT}"

LINUX_BINARY="gomod-linux-x86_64"
DARWIN_BINARY="gomod-darwin-x86_64"
WINDOWS_BINARY="gomod-windows-x86_64.exe"

# Ensure we know which release version we are aiming for.
if [[ -z ${RELEASE_VERSION:-} || ! ${RELEASE_VERSION} =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "Please specify the targeted version via the RELEASE_VERSION environment variable (e.g. '0.5.0')."
	exit 1
fi

TAG="v${RELEASE_VERSION}"

# Ensure we have a GitHub authentication token available.
if [[ -z ${GITHUB_API_TOKEN:-} ]]; then
	echo "Please specify a GitHub API token with appropriate permissions via the GITHUB_API_TOKEN environment variable."
fi

# Ensure the release tag does not yet exist.
if git tag -l | grep --quiet "^${TAG}$"; then
	echo "The targeted releaes '${RELEASE_VERSION}' already seems to exist. Aborting."
	exit 1
fi

printf "\nBuilding release binaries..."
printf "\n- Linux..."
GOARCH=amd64 GOOS=linux go build -o "${LINUX_BINARY}" .
printf " DONE\n- MacOS..."
GOARCH=amd64 GOOS=darwin go build -o "${DARWIN_BINARY}" .
printf " DONE\n- Windows..."
GOARCH=amd64 GOOS=windows go build -o "${WINDOWS_BINARY}" .
echo " DONE"

# Retrieve the release description from the release-notes.
AWK_PROGFILE="$(mktemp)"
echo "BEGIN { state=0 }
{
	if (state == 0 && /^## ${RELEASE_VERSION}$/) {
		state=1
	} else if (state == 1 && /^## [0-9]+\.[0-9]+\.[0-9]+$/) { 
		state=2
	}
	if (state == 1) {
		print \$0
	}
}" >"${AWK_PROGFILE}"
RELEASE_DESCRIPTION="$(awk -f "${AWK_PROGFILE}" "RELEASE_NOTES.md")"
rm -f "${AWK_PROGFILE}"

echo "--- RELEASE DESCRIPTION ---"
echo "${RELEASE_DESCRIPTION}"
echo "--- RELEASE DESCRIPTION ---"
echo ""
echo "Are you sure you want to create the '${RELEASE_VERSION}' release with the description above?"

read -r -p "(Y/n) " -n 1
echo ""
if [[ ! ${REPLY} =~ ^[Yy]$ ]]; then
	echo "Aborting."
	exit 1
fi

RELEASE_NOTES="$(mktemp)"
echo "{
	\"tag_name\": \"${TAG}\",
	\"name\": \"${TAG}\",
	\"body\": \"$(awk '{ printf "%s\\n", $0 }' <<<"${RELEASE_DESCRIPTION//\"/\\\"}")\"
}" >"${RELEASE_NOTES}"

printf "\nTagging and pushing release commit..."
git tag --force "${TAG}"
git push --quiet --force origin "${TAG}"
echo " DONE"

printf "\nCreating the GitHub release..."
CREATE_RESPONSE="$(
	curl --silent \
		--data "@${RELEASE_NOTES}" \
		--header "Authorization: token ${GITHUB_API_TOKEN}" \
		--header "Content-Type: application/json" \
		https://api.github.com/repos/Helcaraxan/gomod/releases
)"

RELEASE_NAME="$(jq --raw-output '.name' <<<"${CREATE_RESPONSE}")"
RELEASE_URL="$(jq --raw-output '.url' <<<"${CREATE_RESPONSE}")"
UPLOAD_URL="$(jq --raw-output '.upload_url' <<<"${CREATE_RESPONSE}")"

if [[ -z ${RELEASE_NAME} || -z ${RELEASE_URL} || -z ${UPLOAD_URL} ]]; then
	echo " FAILED"
	echo ""
	printf "ERROR: It appears that the release creation failed. The API's response was:\n%s\n\n" "${CREATE_RESPONSE}"
	exit 1
fi
echo " DONE"
echo "The release can be found at ${RELEASE_URL}."

echo ""
echo "Uploading release assets..."
RELEASE_ASSETS=(
	"${LINUX_BINARY}"
	"${DARWIN_BINARY}"
	"${WINDOWS_BINARY}"
)
for asset in "${RELEASE_ASSETS[@]}"; do
	echo "- ${asset}..."
	UPLOAD_RESPONSE="$(
		curl --progress-bar \
			--data-binary "@${asset}" \
			--header "Authorization: token ${GITHUB_API_TOKEN}" \
			--header "Content-Type: application/octet-stream" \
			"${UPLOAD_URL%%\{*}?name=${asset}"
	)"
	UPLOAD_STATE="$(jq --raw-output '.state' <<<"${UPLOAD_RESPONSE}")"
	if [[ ${UPLOAD_STATE} != uploaded ]]; then
		echo ""
		printf "ERROR: It appears that the upload of asset ${asset} failed. The API's response was:\n%s\n\n" "${UPLOAD_RESPONSE}"
		exit 1
	fi
done

echo ""
echo "The ${RELEASE_VERSION} release was successfully created."

rm -f "${RELEASE_ASSETS[@]}"
