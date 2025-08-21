/* 

handle text `fmt.Sprintf` like formatting

*/
function typeName(x: any) {
  	if (x === null) return "null";
  	const t = typeof x;
  	if (t !== "object") return t;                // string, number, boolean, bigint, symbol, undefined
  	const ctor = x?.constructor?.name;
  	if (ctor && ctor !== "Object") return ctor;  // Array, Date, Map, CustomClass, ...
  	const tag = Object.prototype.toString.call(x); // "[object Object]"
  	const m = /^\[object (\w+)\]$/.exec(tag);
  	return m ? m[1] : "Object";
}

function defaultRepr(x: any) {
  	if (x === null) return "null";
  	if (x === undefined) return "undefined";
  	const t = typeof x;
  	if (t !== "object") return String(x);
  	// Prefer custom toString if it exists
  	if (x.toString && x.toString !== Object.prototype.toString) return x.toString();
  	try { return JSON.stringify(x); } catch { return Object.prototype.toString.call(x); }
}

function hasFlag(flags: string, f: string) {
	return flags && flags.indexOf(f) !== -1;
}

type fmtFn = (flags: string, width: number | null, precision: number | null, arg: any) => string;

// Signature: (flags, width, precision, arg) -> string
const _fmtMap: Record<string, fmtFn> = {
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
export default function(fmt: string, ...args: any) {
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
