import type { ReactNode } from "react";
import { LoadingState } from "./ui";

export interface Column<T> {
  key: string;
  header: ReactNode;
  render: (row: T) => ReactNode;
  className?: string;
  headerClassName?: string;
  align?: "left" | "right" | "center";
}

interface TableProps<T> {
  columns: Column<T>[];
  rows: T[];
  rowKey: (row: T, index: number) => string;
  onRowClick?: (row: T) => void;
  loading?: boolean;
  loadingLabel?: string;
  empty?: ReactNode;
  /** 不包裹 card 边框（用于已经在 Card 中的表格） */
  bare?: boolean;
  className?: string;
}

const alignClass = { left: "text-left", right: "text-right", center: "text-center" };

export default function Table<T>({
  columns,
  rows,
  rowKey,
  onRowClick,
  loading,
  loadingLabel,
  empty,
  bare,
  className = "",
}: TableProps<T>) {
  const wrapper = bare ? className : `card overflow-hidden ${className}`;

  if (loading && rows.length === 0) {
    return (
      <div className={wrapper}>
        <LoadingState label={loadingLabel} />
      </div>
    );
  }

  if (rows.length === 0) {
    return <div className={wrapper}>{empty}</div>;
  }

  return (
    <div className={wrapper}>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="border-b border-line bg-canvas/60">
              {columns.map((column) => (
                <th
                  key={column.key}
                  scope="col"
                  className={`whitespace-nowrap px-4 py-2.5 text-xs font-medium text-fg-subtle first:pl-5 last:pr-5 ${alignClass[column.align || "left"]} ${column.headerClassName || ""}`}
                >
                  {column.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-line">
            {rows.map((row, index) => (
              <tr
                key={rowKey(row, index)}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
                className={`group transition-colors hover:bg-subtle/60 ${onRowClick ? "cursor-pointer" : ""}`}
              >
                {columns.map((column) => (
                  <td
                    key={column.key}
                    className={`px-4 py-3 align-middle text-fg first:pl-5 last:pr-5 ${alignClass[column.align || "left"]} ${column.className || ""}`}
                  >
                    {column.render(row)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
