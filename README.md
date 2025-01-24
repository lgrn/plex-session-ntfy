# plex-session-ntfy

After setting relevant variables in `psn.py`, run it to monitor the Plex
API for any new sessions. When one is found, a notification will be sent
to the topic configured.

This is just a proof of concept, and there are many things to improve.
PRs are welcome.

## How to run

Clone the repo, set up a virtual environment with `python -m venv`,
source `bin/activate` and run `psn.py`.

You may want to run this as a service.
