#!/bin/bash
# Git askpass helper - provides authentication token to git
# Used for HTTPS authentication with GitHub/GitLab tokens

# When git needs credentials, it calls this script with a prompt as $1
# We respond with the appropriate credential based on the prompt

case "$1" in
    Username*)
        # For GitHub, username can be anything (token is what matters)
        # For GitLab, username should be "oauth2"
        if [[ "${GIT_REPO:-}" == *"gitlab"* ]]; then
            echo "oauth2"
        else
            echo "x-access-token"
        fi
        ;;
    Password*)
        # Provide the token as the password
        echo "${GIT_TOKEN}"
        ;;
    *)
        # Unknown prompt - provide token anyway
        echo "${GIT_TOKEN}"
        ;;
esac
