#!/usr/bin/env bash
# vim: set tabstop=2 shiftwidth=2 expandtab
set -e -u -o pipefail

readonly project_root="$(dirname "${BASH_SOURCE[0]}")/.."
cd "${project_root}"

readonly linux_binary="gomod-linux-x86_64"
readonly darwin_binary="gomod-darwin-x86_64"
readonly windows_binary="gomod-windows-x86_64.exe"

# Ensure we know which release version we are aiming for.
if [[ -z ${RELEASE_VERSION:-} || ! ${RELEASE_VERSION} =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Please specify the targeted version via the RELEASE_VERSION environment variable (e.g. '0.5.0')."
  exit 1
fi

readonly tag="v${RELEASE_VERSION}"

# Ensure we have a GitHub authentication token available.
if [[ -z ${GITHUB_API_TOKEN:-} ]]; then
  echo "Please specify a GitHub API token with appropriate permissions via the GITHUB_API_TOKEN environment variable."
  exit 1
fi

# Ensure the release tag does not yet exist.
if git tag -l | grep --quiet "^${tag}$"; then
  echo "The targeted releaes '${RELEASE_VERSION}' already seems to exist. Aborting."
  exit 1
fi

readonly build_time="$(date -u +'%Y-%m-%d %H:%M:%S')"

printf "\nBuilding release binaries..."
printf "\n- Linux..."
GOARCH=amd64 GOOS=linux go build -o "${linux_binary}" -ldflags "-X 'main.toolVersion=${tag}' -X 'main.toolDate=${build_time}'" .
printf " DONE\n- MacOS..."
GOARCH=amd64 GOOS=darwin go build -o "${darwin_binary}" -ldflags "-X 'main.toolVersion=${tag}' -X 'main.toolDate=${build_time}'" .
printf " DONE\n- Windows..."
GOARCH=amd64 GOOS=windows go build -o "${windows_binary}" -ldflags "-X 'main.toolVersion=${tag}' -X 'main.toolDate=${build_time}'" .
echo " DONE"

# Retrieve the release description from the release-notes.
readonly awk_progfile="$(mktemp)"
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
}" >"${awk_progfile}"
readonly release_description="$(awk -f "${awk_progfile}" "RELEASE_NOTES.md")"
rm -f "${awk_progfile}"

echo "--- RELEASE DESCRIPTION ---"
echo "${release_description}"
echo "--- RELEASE DESCRIPTION ---"
echo ""
echo "Are you sure you want to create the '${RELEASE_VERSION}' release with the description above?"

read -r -p "(Y/n) " -n 1
echo ""
if [[ ! ${REPLY} =~ ^[Yy]$ ]]; then
  echo "Aborting."
  exit 1
fi

readonly release_notes="$(mktemp)"
echo "{
	\"tag_name\": \"${tag}\",
	\"name\": \"${tag}\",
	\"body\": \"$(awk '{ printf "%s\\n", $0 }' <<<"${release_description//\"/\\\"}")\"
}" >"${release_notes}"

printf "\nTagging and pushing release commit..."
git tag --force "${tag}"
git push --quiet --force origin "${tag}"
echo " DONE"

printf "\nCreating the GitHub release..."
readonly create_response="$(
  curl --silent \
    --data "@${release_notes}" \
    --header "Authorization: token ${GITHUB_API_TOKEN}" \
    --header "Content-Type: application/json" \
    https://api.github.com/repos/Helcaraxan/gomod/releases
)"

readonly release_name="$(jq --raw-output '.name' <<<"${create_response}")"
readonly release_url="$(jq --raw-output '.url' <<<"${create_response}")"
readonly upload_url="$(jq --raw-output '.upload_url' <<<"${create_response}")"

if [[ -z ${release_name} || -z ${release_url} || -z ${upload_url} ]]; then
  echo " FAILED"
  echo ""
  printf "ERROR: It appears that the release creation failed. The API's response was:\n%s\n\n" "${create_response}"
  exit 1
fi
echo " DONE"
echo "The release can be found at ${release_url}."

echo ""
echo "Uploading release assets..."
readonly release_assets=(
  "${linux_binary}"
  "${darwin_binary}"
  "${windows_binary}"
)
for asset in "${release_assets[@]}"; do
  echo "- ${asset}..."
  readonly upload_response="$(
    curl --progress-bar \
      --data-binary "@${asset}" \
      --header "Authorization: token ${GITHUB_API_TOKEN}" \
      --header "Content-Type: application/octet-stream" \
      "${upload_url%%\{*}?name=${asset}"
  )"
  readonly upload_state="$(jq --raw-output '.state' <<<"${upload_response}")"
  if [[ ${upload_state} != uploaded ]]; then
    echo ""
    printf "ERROR: It appears that the upload of asset ${asset} failed. The API's response was:\n%s\n\n" "${upload_response}"
    exit 1
  fi
  unset upload_response
done

echo ""
echo "The ${RELEASE_VERSION} release was successfully created."

rm -f "${release_assets[@]}"
