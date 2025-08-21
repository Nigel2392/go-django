(function() {

if (!window.sprintf) {
	/* 

	handle text `fmt.Sprintf` like formatting

	*/
	function typeName(x) {
	  	if (x === null) return "null";
	  	const t = typeof x;
	  	if (t !== "object") return t;                // string, number, boolean, bigint, symbol, undefined
	  	const ctor = x?.constructor?.name;
	  	if (ctor && ctor !== "Object") return ctor;  // Array, Date, Map, CustomClass, ...
	  	const tag = Object.prototype.toString.call(x); // "[object Object]"
	  	const m = /^\[object (\w+)\]$/.exec(tag);
	  	return m ? m[1] : "Object";
	}

	function defaultRepr(x) {
	  	if (x === null) return "null";
	  	if (x === undefined) return "undefined";
	  	const t = typeof x;
	  	if (t !== "object") return String(x);
	  	// Prefer custom toString if it exists
	  	if (x.toString && x.toString !== Object.prototype.toString) return x.toString();
	  	try { return JSON.stringify(x); } catch { return Object.prototype.toString.call(x); }
	}

	function hasFlag(flags, f) {
		return flags && flags.indexOf(f) !== -1;
	}

	// Signature: (flags, width, precision, arg) -> string
	const _fmtMap = {
	  	// %v : default representation
	  	v(flags, _w, _p, arg) {
	  	  	if (hasFlag(flags, "#")) {  // %#v -> JSON-ish
	  	  	  	try { return JSON.stringify(arg); } catch { return defaultRepr(arg); }
	  	  	}
	  	  	return defaultRepr(arg);
	  	},

	  	// %s : string (optional precision => max length)
	  	s(_flags, _w, precision, arg) {
	  	  	let s = String(arg);
	  	  	if (precision != null) s = s.slice(0, precision);
	  	  	return s;
	  	},

		// %q : quoted string
		q(_flags, _w, _p, arg) {
		  	return `"${String(arg).replace(/"/g, '\\"')}"`;
		},

	  	// %d : integer (honor '+' sign flag)
	  	d(flags, _w, _p, arg) {
	  	  	let n = Number.parseInt(arg, 10);
	  	  	if (!Number.isFinite(n)) n = 0;
	  	  	const sign = n < 0 ? "-" : (hasFlag(flags, "+") ? "+" : "");
	  	  	return sign + Math.abs(n).toString(10);
	  	},

	  	// %f : float with precision (default 6), honor '+' sign flag
	  	f(flags, _w, precision, arg) {
	  	  	let n = Number.parseFloat(arg);
	  	  	if (!Number.isFinite(n)) n = 0;
	  	  	const p = precision != null ? precision : 6;
	  	  	const body = Math.abs(n).toFixed(p);
	  	  	const sign = n < 0 ? "-" : (hasFlag(flags, "+") ? "+" : "");
	  	  	return sign + body;
	  	},

	  	// %T : type name (Go-esque)
	  	T(_flags, _w, _p, arg) {
	  	  	return typeName(arg);
	  	}
	};

	// Supports: %% literal percent
	// Verb set: s T d f v
	// Flags (minimal): '+' (numbers), '#' (for %v -> JSON)
	// Precision: only for %f (e.g., %.2f) and %s (truncate)
	function sprintf(fmt, ...args) {
		if (args.length === 0) {
			return fmt;
		}

	  	let i = 0;
	  	// %%% OR % [flags]* [width]? (.precision)? [verb]
	  	// We parse flags (+#), optional width (ignored here), optional .precision, then verb.
	  	const re = /%(%|([+#]*)(\d+)?(?:\.(\d+))?([sTdfvq]))/g;

	  	return fmt.replace(re, (_m, pct, flags, widthStr, precStr, verb) => {
	  	  	if (pct === "%") return "%"; // "%%"
	  	  	const width = widthStr ? parseInt(widthStr, 10) : null;      // parsed but unused
	  	  	const precision = precStr != null ? parseInt(precStr, 10) : null;

	  	  	const fn = _fmtMap[verb];
	  	  	if (!fn) throw new Error(`unknown format specifier %${verb}`);

	  	  	const arg = args[i++];
	  	  	return fn(flags || "", width, precision, arg);
	  	});
	}

	window.sprintf = sprintf;
}

/* 

handle text translations

*/


// map[string][]string
let _translations = {{ .Data.translations | json }};

// FNV-1a 32-bit over UTF-8 bytes
function fnv32aBytes(bytes, seed = 0x811c9dc5) {
  	let h = seed >>> 0;
  	for (let i = 0; i < bytes.length; i++) {
  	  	h ^= bytes[i];
  	  	h = Math.imul(h, 0x01000193) >>> 0; // FNV prime
  	}
  	return h >>> 0;
}

// This function should provide the same output as the Go fnv.New32a sum.
function fnv32aHashForPlural(singular, plural) {
  	const enc = new TextEncoder();                 // UTF-8
  	let h = fnv32aBytes(enc.encode(singular));     // start from offset basis
  	h = fnv32aBytes(enc.encode(plural), h);        // continue the stream
  	return h.toString(10);                         // decimal string like Go
}

function pluralidx(num) {
    if (Array.isArray(num)) {
        num = num.length;
    }

    let n = num;
    var v = {{ .Data.header.PluralRule }};
    return (typeof(v) == 'boolean') ? (v ? 1 : 0) : v;
}

function _gettext(key) {
    var translation = _translations[key];
    if (translation === undefined) {
        // Fallback to the key itself if no translation is found
        return key;
    }

    // Return the singular form
    return translation[0];
}

function _ngettext(singular, plural, n) {
    let key = fnv32aHashForPlural(singular, plural);
    let pluralIndex = pluralidx(n);
    let translation = _translations[key];
    if (translation === undefined) {
        if (pluralIndex === 0) {
            return singular;
        }
        return plural;
    }

    if (pluralIndex < translation.length) {
        return translation[pluralIndex];
    }

    if (pluralIndex === 0) {
        return singular;
    }

    return plural;
}

function gettext(key, ...args) {
	return window.sprintf(_gettext(key), ...args);
}

function ngettext(singular, plural, n, ...args) {
	return window.sprintf(_ngettext(singular, plural, n), ...args);
}

/* 

setup global window object

*/

window.i18n = {
    gettext: gettext,
    ngettext: ngettext,
    pluralidx: pluralidx,
    translations: _translations,
    header: {{ .Data.header | json }},

	debug: {
	    fnv32aHashForPlural: fnv32aHashForPlural,
		sprintf: window.sprintf,
		typeName: typeName,
		defaultRepr: defaultRepr,
		_gettext: _gettext,
		_ngettext: _ngettext,
	}
};

})();
