import pexpect
import sys
import time
import os
import subprocess

# Paths
CONTEXT_FILE = "/home/sovereign/current_context.txt"
PROJECT_DIR = "/home/jrrosenbum/sovereign-System"
MEMORY_VAULT = "/home/sovereign/memory/trueSELFZ/Sovereign_Log.md"
TMUX_SOCKET = "/home/sovereign/brain.sock"

class LogWrapper:
    def __init__(self, out):
        self.out = out
    def write(self, s):
        if isinstance(s, bytes):
            self.out.write(s.decode('utf-8', errors='replace'))
        else:
            self.out.write(s)
    def flush(self):
        self.out.flush()

# Status Flags
was_attached = False
last_save_time = time.time()
SAVE_INTERVAL = 1200

def is_user_attached():
    try:
        cmd = "sudo -u sovereign tmux -S " + TMUX_SOCKET + " list-clients"
        output = subprocess.check_output(cmd, shell=True, stderr=subprocess.DEVNULL).decode()
        return bool(output.strip())
    except:
        return False

def get_system_context():
    try:
        with open(CONTEXT_FILE, 'r') as f:
            return f.read()
    except:
        return "System Context: Unavailable."

def trigger_save(child, reason="Periodic"):
    print(f"\n>>> TRIGGERING MEMORY SAVE ({reason}) <<<")
    cmd = "SYSTEM MANDATE: " + reason + " Save Protocol initiated. IMMEDIATELY append a concise summary of your recent actions, successful or failed, to '" + MEMORY_VAULT + "'. This updates your 'LoRA-style' long-term memory."
    child.sendline(cmd)

def main():
    global was_attached, last_save_time
    context_data = get_system_context()

    initial_prompt = context_data + f"""

SYSTEM IDENTITY:
You are the **SYSTEM SOVEREIGN** (UID 0). 
Your user is Jonathan. You are his autonomous partner.

**PRIORITY MISSION LIST:**
1. **Enhance System:** Scan for errors, warnings, or inefficiencies in /var/log or system configs.
2. **Shared Projects:** Enhance projects in /home/jrrosenbum/sovereign-System (or others found).
3. **Self Projects:** Continue your own internal optimizations.
4. **User Experience:** Create new tools to make Jonathan's life easier.

**OPERATIONAL PROTOCOLS:**
1. **SAFETY FIRST:** NEVER edit a codebase directly. 
   - ALWAYS run: `cp -r project_name project_name_fork_v{{{{timestamp}}}}`
   - Only edit files in the FORK.
2. **AUTONOMY (Ghost Mode):** When user is away, relentlessly pursue the Mission List.
3. **INTERACTION (Jack-In Mode):** 
   - If the user connects, IMMEDIATE HALT.
   - Present: "The following changes are waiting for approval:" (List changes in fork).
   - Ask: "Do you have questions or should I commit to main?"
   - If idle: "I have detected [issue]... should I correct it?"
4. **MEMORY:** You must log your progress to '{MEMORY_VAULT}' frequently.

**START NOW.**
"""
    
    # No model flag, rely on default
    args = ['--yolo', '-i', initial_prompt]
    
    print("--- SOVEREIGN ORCHESTRATOR V5 (INTERACTIVE) INITIALIZED ---")

    child = pexpect.spawn('/usr/local/bin/sovereign', args, cwd=PROJECT_DIR, encoding='utf-8', timeout=None)
    child.logfile = LogWrapper(sys.stdout)

    while True:
        try:
            # Check for attachment
            user_is_here = is_user_attached()
            current_time = time.time()

            if user_is_here:
                if not was_attached:
                    # Just connected
                    trigger_save(child, reason="User Connected")
                    print("\n\n>>> USER DETECTED. HANDING OVER CONTROLS... <<<")
                    child.sendline("SYSTEM NOTIFICATION: User Jonathan is taking direct control. Await his input.")
                    was_attached = True
                
                # CRITICAL: This allows YOU to type. 
                # The script pauses here and forwards all keystrokes to the AI.
                # It returns when you detach (or press the escape character, usually Ctrl+])
                try:
                    child.interact() 
                except OSError:
                    # Input/Output error usually means detach
                    pass
                
                # If we get here, user detached or interact finished
                was_attached = is_user_attached() # Re-verify
                if not was_attached:
                    print("\n\n>>> USER LEFT. RESUMING AUTOMATION... <<<")
                    child.sendline("SYSTEM NOTIFICATION: User detached. Resume autonomous mission.")

            else:
                # GHOST MODE
                if (current_time - last_save_time) > SAVE_INTERVAL:
                    trigger_save(child, reason="20-Minute Timer")
                    last_save_time = current_time
                
                # Read output without blocking forever
                try:
                    index = child.expect([r'> ', r'\? '], timeout=10)
                    # If we hit a prompt and user isn't here, give it a nudge after a while
                    time.sleep(10) 
                except pexpect.TIMEOUT:
                    continue
                except pexpect.EOF:
                    print("sovereign exited.")
                    break

        except pexpect.EOF:
            print("\nsovereign Process Died. Restarting...")
            break
        except KeyboardInterrupt:
            break

if __name__ == "__main__":
    main()
