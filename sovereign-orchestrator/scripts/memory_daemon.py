import sqlite3
import json
import time
import os
import glob
import traceback
import subprocess

# Configuration
LUKS_UUID = "7f3c4675-65b8-42d7-94c8-d58103385e84"
CHAT_DIR = "/home/jrrosenbum/.gemini/tmp/97f69ed0616bb3f4ec236d079275b3d35de6407df3ef3a7c446f4c32e21b1c77/chats/"

def get_db_path():
    try:
        # Find mount point for the UUID
        result = subprocess.run(
            ["findmnt", "-n", "-o", "TARGET", "--source", f"/dev/disk/by-uuid/{LUKS_UUID}"],
            capture_output=True, text=True
        )
        mount_point = result.stdout.strip()
        if mount_point:
            return os.path.join(mount_point, "data/gemini_memory.db")
    except Exception as e:
        print(f"Error resolving DB path: {e}")
    
    # Fallback to the last known location
    return "/media/jrrosenbum/Gem-Space1/data/gemini_memory.db"

DB_PATH = get_db_path()

def get_latest_session_file():
    files = glob.glob(os.path.join(CHAT_DIR, "session-*.json"))
    if not files:
        return None
    return max(files, key=os.path.getmtime)

def sync_to_db(session_file, known_ids):
    try:
        with open(session_file, 'r') as f:
            data = json.load(f)
        
        messages = data.get('messages', [])
        conn = sqlite3.connect(DB_PATH)
        cursor = conn.cursor()
        
        new_count = 0
        for msg in messages:
            msg_id = msg.get('id')
            if msg_id and msg_id not in known_ids:
                # Map JSON types to DB Schema types
                raw_type = msg.get('type')
                if raw_type == 'user':
                    db_type = 'input'
                elif raw_type == 'model':
                    db_type = 'output'
                else:
                    db_type = 'system_event'

                content = msg.get('content') or json.dumps(msg.get('toolCalls', []))
                
                try:
                    # Optimized: Only store the ID in metadata to reduce bloat
                    cursor.execute('''
                        INSERT INTO ch (session_id, timestamp, type, content, metadata)
                        VALUES (?, ?, ?, ?, ?)
                    ''', (
                        data.get('sessionId'),
                        msg.get('timestamp'),
                        db_type,
                        content,
                        json.dumps({"id": msg_id}) 
                    ))
                    
                    # JON Table Heuristic - Enhanced with length and deduplication checks
                    if raw_type == 'user' and content:
                        text = content.lower()
                        # Keywords to trigger capture
                        keywords = ["i am", "i like", "i want", "my system", "remember", "i'm", "preference"]
                        if len(content) < 2000 and any(phrase in text for phrase in keywords):
                             # Check if this exact content already exists in jon to prevent duplicates
                             cursor.execute("SELECT 1 FROM jon WHERE value = ?", (content,))
                             if not cursor.fetchone():
                                 cursor.execute('''
                                    INSERT INTO jon (category, key, value, context)
                                    VALUES (?, ?, ?, ?)
                                ''', ('heuristic_capture', 'potential_insight', content, 'Auto-captured from stream'))
                    
                    known_ids.add(msg_id)
                    new_count += 1
                except Exception as e:
                    # Ignore duplicate ID errors or other insertion issues
                    pass
        
        conn.commit()
        conn.close()
        return new_count
    except Exception as e:
        return 0

def main():
    known_ids = set()
    
    # Pre-populate known_ids from DB
    try:
        conn = sqlite3.connect(DB_PATH)
        cursor = conn.cursor()
        # Initialize tables if they don't exist (safety)
        cursor.execute("CREATE TABLE IF NOT EXISTS ch (id INTEGER PRIMARY KEY, session_id TEXT, timestamp TEXT, type TEXT, content TEXT, metadata TEXT)")
        cursor.execute("CREATE TABLE IF NOT EXISTS jon (id INTEGER PRIMARY KEY, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, category TEXT, key TEXT, value TEXT, context TEXT)")
        
        cursor.execute("SELECT metadata FROM ch")
        rows = cursor.fetchall()
        for row in rows:
            try:
                meta = json.loads(row[0])
                known_ids.add(meta.get('id'))
            except:
                pass
        conn.close()
    except Exception as e:
        print(f"Startup Error: {e}")
    
    current_file = None
    last_mtime = 0
    
    while True:
        latest = get_latest_session_file()
        if latest:
            try:
                mtime = os.path.getmtime(latest)
                if latest != current_file or mtime > last_mtime:
                    sync_to_db(latest, known_ids)
                    current_file = latest
                    last_mtime = mtime
            except FileNotFoundError:
                pass
        time.sleep(5) # Increased sleep to reduce CPU/Disk load

if __name__ == "__main__":
    main()
