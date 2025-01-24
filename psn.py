import requests
import time

# CONFIGURATION

PLEX_IP = "127.0.0.1"
# PLEX_TOKEN can be fetched by finding PlexOnlineToken in
# Preferences.xml, for example with this command:
# grep -oP 'PlexOnlineToken="\K[^"]+' \
# "/var/lib/plexmediaserver/Library/Application Support/Plex Media Server/Preferences.xml"
PLEX_TOKEN = ""
NTFY_TOPIC_URL = "https://ntfy.sh/your-topic-here"
# CHECK_INTERVAL sets how often the API is queried for new sessions
CHECK_INTERVAL = 30

# notified_sessions tracks previously notified sessions
notified_sessions = {}

def fetch_sessions():
    """
    Fetch current sessions from the Plex server.
    """
    try:
        url = f"http://{PLEX_IP}:32400/status/sessions?X-Plex-Token={PLEX_TOKEN}"
        response = requests.get(url)
        if response.status_code == 200:
            return response.text
        else:
            print(f"Failed to fetch sessions: {response.status_code}, {response.text}")
            return None
    except Exception as e:
        print(f"Error fetching sessions: {e}")
        return None

def parse_sessions(xml_data):
    """
    Parse the XML response to extract session details.
    """
    import xml.etree.ElementTree as ET

    try:
        root = ET.fromstring(xml_data)
        sessions = []
        for video in root.findall("Video"):
            # if you want to include something else, just inspect the
            # response you get from the API and add it here
            session_data = {
                "sessionKey": video.get("sessionKey"),
                "user": video.find("User").get("title"),
                "device": video.find("Player").get("title"),
                "devicePlatform": video.find("Player").get("platform"),
                "deviceState": video.find("Player").get("state"),
                "grandparentTitle": video.get("grandparentTitle"),
                "mediaTitle": video.get("title"),
                "mediaType": video.get("type"),
                "mediaKey": video.get("key"),
            }
            sessions.append(session_data)
        return sessions
    except Exception as e:
        print(f"Error parsing XML: {e}")
        return []

def send_notification(session):
    """
    Send a notification for a new session.
    """
    # if this is a movie, grandparent is "None", fix that:
    if session['grandparentTitle'] == None:
        session['grandparentTitle'] = "(film)"

    payload = f"{session['user']} {session['grandparentTitle']}: '{session['mediaTitle']}' on {session['device']} ({session['devicePlatform']})"
    try:
        response = requests.post(NTFY_TOPIC_URL,
            data = payload,
            headers={
                "Title": "Plex session active",
                "Tags": "satellite"
            })
        if response.status_code == 200:
            print(f"Notification sent for session: {session['user']}")
        else:
            print(f"Failed to send notification: {response.status_code}, {response.text}")
    except Exception as e:
        print(f"Error sending notification: {e}")

def monitor_sessions():
    """
    Monitor the Plex server for new sessions and send notifications.
    """
    global notified_sessions

    while True:
        xml_data = fetch_sessions()
        if not xml_data:
            time.sleep(CHECK_INTERVAL)
            continue

        current_sessions = parse_sessions(xml_data)

        for session in current_sessions:
            session_key = session["sessionKey"]
            if session_key not in notified_sessions:
                send_notification(session)
                notified_sessions[session_key] = time.time()
            else:
                print("Skipping notification.")

        # Remove sessions that are no longer active
        active_keys = {s["sessionKey"] for s in current_sessions}
        notified_sessions = {k: v for k, v in notified_sessions.items() if k in active_keys}

        time.sleep(CHECK_INTERVAL)

if __name__ == "__main__":
    monitor_sessions()
