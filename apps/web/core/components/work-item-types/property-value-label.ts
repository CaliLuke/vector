export function scalarLabel(value: unknown) {
  if (typeof value === "boolean") return value ? "Yes" : "No";
  return typeof value === "string" || typeof value === "number" ? String(value) : "—";
}

export function dateLabel(value: string, format?: string) {
  const parts = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!parts) return value;
  const [, year, month, day] = parts;
  if (format === "15/01/2025") return `${day}/${month}/${year}`;
  if (format === "01/15/2025") return `${month}/${day}/${year}`;
  if (format === "2025/01/15") return `${year}/${month}/${day}`;
  const monthName = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"][
    Number(month) - 1
  ];
  return monthName ? `${monthName} ${Number(day)}, ${year}` : value;
}
