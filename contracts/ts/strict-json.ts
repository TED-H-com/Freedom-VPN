// Independent strict parser for the shared vector verifier. JSON.parse alone
// loses duplicate members and the original spelling of numbers.
export type Json = null | boolean | number | string | Json[] | { [k: string]: Json };

export function parseStrictJSON(raw: Uint8Array): Json {
  const fail = (code: string): never => { throw new Error(code); };
  if (raw.byteLength > 1 << 20) fail("too_large");
  let text: string;
  try {
    // Preserve the BOM so it is rejected as syntax, not silently removed.
    text = new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(raw);
  } catch { return fail("invalid_utf8"); }
  let pos = 0;
  const ws = (): void => { while (pos < text.length && /[ \t\r\n]/.test(text[pos])) pos++; };
  const string = (): string => {
    const start = pos++;
    let closed = false;
    while (pos < text.length) {
      const ch = text[pos++];
      if (ch === '"') { closed = true; break; }
      if (ch === "\\") pos++;
    }
    if (!closed) fail("syntax");
    let value: string;
    try { value = JSON.parse(text.slice(start, pos)); } catch { return fail("syntax"); }
    for (let i = 0; i < value.length; i++) {
      const c = value.charCodeAt(i);
      if (c >= 0xd800 && c <= 0xdbff) {
        const next = value.charCodeAt(++i);
        if (!(next >= 0xdc00 && next <= 0xdfff)) fail("lone_surrogate");
      } else if (c >= 0xdc00 && c <= 0xdfff) fail("lone_surrogate");
    }
    return value;
  };
  const value = (depth: number): Json => {
    ws();
    const ch = text[pos];
    if (ch === '"') return string();
    if (ch === "{" || ch === "[") {
      if (depth >= 256) fail("too_deep");
      pos++;
      const object = ch === "{";
      const end = object ? "}" : "]";
      const result: { [k: string]: Json } = Object.create(null);
      const array: Json[] = [];
      const keys = new Set<string>();
      ws();
      if (text[pos] === end) { pos++; return object ? result : array; }
      while (true) {
        ws();
        if (object) {
          if (text[pos] !== '"') fail("syntax");
          const key = string();
          if (keys.has(key)) fail("duplicate_key");
          keys.add(key);
          ws();
          if (text[pos++] !== ":") fail("syntax");
          result[key] = value(depth + 1);
        } else array.push(value(depth + 1));
        ws();
        if (text[pos] === end) { pos++; return object ? result : array; }
        if (text[pos++] !== ",") fail("syntax");
      }
    }
    for (const [literal, parsed] of [["null", null], ["true", true], ["false", false]] as const) {
      if (text.startsWith(literal, pos)) { pos += literal.length; return parsed; }
    }
    const match = /^-?(?:0|[1-9][0-9]*)/.exec(text.slice(pos));
    if (!match) return fail("syntax");
    const number = match[0];
    pos += number.length;
    if (pos < text.length && /[.eE]/.test(text[pos])) fail("non_integer");
    if (number === "-0") fail("negative_zero");
    const parsed = Number(number);
    if (!Number.isSafeInteger(parsed)) fail("integer_range");
    return parsed;
  };
  const parsed = value(0);
  ws();
  if (pos !== text.length) fail("syntax");
  return parsed;
}
