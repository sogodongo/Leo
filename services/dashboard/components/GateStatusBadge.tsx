"use client";
export function GateStatusBadge({ state }: { state: "open"|"closed"|"pending" }) {
  const s = {open:"color:#4ade80",closed:"color:#f87171",pending:"color:#fbbf24"};
  const l = {open:"✓ open",closed:"✗ blocked",pending:"⠋ running"};
  return <span style={{fontSize:12,fontFamily:"monospace",...(s[state] as any)}}>{l[state]}</span>;
}
