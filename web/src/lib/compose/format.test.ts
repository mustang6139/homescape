import { describe, it, expect } from "vitest";
import { formatValue, relTime } from "./format";

describe("formatValue", () => {
  it("ms / percent / round", () => {
    expect(formatValue(12.6, "ms")).toBe("13ms");
    expect(formatValue(64.1, "percent")).toBe("64%");
    expect(formatValue(2.7, "round")).toBe("3");
  });

  it("bytes scales to binary units", () => {
    expect(formatValue(0, "bytes")).toBe("0 B");
    expect(formatValue(1024, "bytes")).toBe("1.0 KB");
    expect(formatValue(11.4 * 1024 ** 3, "bytes")).toBe("11.4 GB");
    expect(formatValue(2 * 1024 ** 4, "bytes")).toBe("2.0 TB");
  });

  it("duration is compact", () => {
    expect(formatValue(45, "duration")).toBe("45s");
    expect(formatValue(125, "duration")).toBe("2m 5s");
    expect(formatValue(3700, "duration")).toBe("1h 1m");
    expect(formatValue(90000, "duration")).toBe("1d 1h");
  });

  it("text and default stringify, null → empty", () => {
    expect(formatValue("hi", "text")).toBe("hi");
    expect(formatValue(42)).toBe("42");
    expect(formatValue(null)).toBe("");
    expect(formatValue(undefined, "ms")).toBe("");
  });

  it("non-numeric input to numeric presets → empty", () => {
    expect(formatValue("nope", "ms")).toBe("");
    expect(formatValue({}, "percent")).toBe("");
  });
});

describe("relTime", () => {
  const now = 1_700_000_000_000; // fixed reference (2023, realistic epoch ms)
  it("renders seconds/minutes/hours/days ago", () => {
    expect(relTime(now - 5_000, now)).toBe("5s ago");
    expect(relTime(now - 5 * 60_000, now)).toBe("5m ago");
    expect(relTime(now - 3 * 3_600_000, now)).toBe("3h ago");
    expect(relTime(now - 2 * 86_400_000, now)).toBe("2d ago");
  });

  it("parses ISO strings", () => {
    const iso = new Date(now - 10_000).toISOString();
    expect(relTime(iso, now)).toBe("10s ago");
  });

  it("future → just now, invalid → empty", () => {
    expect(relTime(now + 5000, now)).toBe("just now");
    expect(relTime("not a date", now)).toBe("");
  });
});
