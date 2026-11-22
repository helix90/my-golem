# Fetches the AIML repository named by AIML_REPO into /aiml, which the bot
# mounts at /app/testdata/zz-local. Run by the aiml-sync service in
# docker-compose.yml before the bot starts.
#
#   AIML_REPO   repository URL; unset means no extra AIML
#   AIML_REF    branch or tag; default is the repository's default branch
#   AIML_TOKEN  token for a private HTTPS repository (GitHub: a fine-grained
#               token with read-only Contents access). It is sent with each
#               fetch and not stored in the clone.
set -e
cd /aiml

if [ -z "$AIML_REPO" ]; then
  # Remove AIML from a repository that is no longer configured
  find . -mindepth 1 -delete
  echo "AIML_REPO not set; no extra AIML"
  exit 0
fi

git config --global --add safe.directory '*'
[ -d .git ] || git init -q .

auth=""
if [ -n "$AIML_TOKEN" ]; then
  auth="http.extraHeader=Authorization: Basic $(printf 'x-access-token:%s' "$AIML_TOKEN" | base64 | tr -d '\n')"
fi

if git ${auth:+-c "$auth"} fetch -q --depth 1 "$AIML_REPO" "${AIML_REF:-HEAD}"; then
  git reset -q --hard FETCH_HEAD
  git clean -qfdx
  echo "AIML at $(git log -1 --format='%h %s')"
elif git rev-parse -q --verify HEAD >/dev/null; then
  # Keep the bot running on the last copy when the repository can't be reached
  echo "Could not fetch $AIML_REPO; keeping $(git log -1 --format='%h %s')"
else
  echo "Could not fetch $AIML_REPO" >&2
  exit 1
fi
