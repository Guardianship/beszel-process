import { t } from "@lingui/core/macro"
import { Trans } from "@lingui/react/macro"
import {
	type ColumnFiltersState,
	flexRender,
	getCoreRowModel,
	getFilteredRowModel,
	getSortedRowModel,
	type Row,
	type SortingState,
	type Table as TableType,
	useReactTable,
	type VisibilityState,
} from "@tanstack/react-table"
import { useVirtualizer, type VirtualItem } from "@tanstack/react-virtual"
import { memo, useEffect, useMemo, useRef, useState } from "react"
import { Card, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { pb } from "@/lib/api"
import { $allSystemsById } from "@/lib/stores"
import { cn, useBrowserStorage } from "@/lib/utils"
import type { ProcessRecord } from "@/types"
import { processTableCols } from "./process-table-columns"

export default function ProcessTable({ systemId }: { systemId?: string }) {
	const loadTime = Date.now()
	const [data, setData] = useState<ProcessRecord[]>([])
	const [sorting, setSorting] = useBrowserStorage<SortingState>(
		`sort-proc-${systemId ? 1 : 0}`,
		[{ id: systemId ? "name" : "system", desc: false }],
		sessionStorage
	)
	const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])
	const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})
	const [globalFilter, setGlobalFilter] = useState("")

	useEffect(() => {
		return setData([])
	}, [systemId])

	useEffect(() => {
		function fetchData(systemId?: string) {
			pb.collection<ProcessRecord>("processes")
				.getList(0, 2000, {
					fields: "name,pid,cpu,memory,status,uptime,updated",
					filter: systemId ? pb.filter("system={:system}", { system: systemId }) : undefined,
				})
				.then(
					({ items }) =>
						items.length &&
						setData((curItems) => {
							const processNames = new Set<string>()
							const newItems: ProcessRecord[] = []
							for (const item of items) {
								processNames.add(item.name)
								newItems.push(item)
							}
							for (const item of curItems) {
								if (!processNames.has(item.name)) {
									newItems.push(item)
								}
							}
							return newItems
						})
				)
		}

		fetchData(systemId)

		if (!systemId) {
			return $allSystemsById.listen((_value, _oldValue, systemId) => {
				if (Date.now() - loadTime > 500) {
					fetchData(systemId)
				}
			})
		}

		return () => {}
	}, [systemId])

	const table = useReactTable({
		data,
		columns: processTableCols,
		getCoreRowModel: getCoreRowModel(),
		getSortedRowModel: getSortedRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		onSortingChange: setSorting,
		onColumnFiltersChange: setColumnFilters,
		onColumnVisibilityChange: setColumnVisibility,
		defaultColumn: {
			sortUndefined: "last",
			size: 100,
			minSize: 0,
		},
		state: {
			sorting,
			columnFilters,
			columnVisibility,
			globalFilter,
		},
		onGlobalFilterChange: setGlobalFilter,
		globalFilterFn: (row, _columnId, filterValue) => {
			const proc = row.original
			const systemName = $allSystemsById.get()[proc.system]?.name ?? ""
			const name = proc.name ?? ""
			const status = proc.status ?? ""
			const searchString = `${systemName} ${name} ${status}`.toLowerCase()
			return (filterValue as string)
				.toLowerCase()
				.split(" ")
				.every((term) => searchString.includes(term))
		},
	})

	const rows = table.getRowModel().rows
	const visibleColumns = table.getVisibleLeafColumns()

	const runningCount = useMemo(() => data.filter((p) => p.status !== "stopped").length, [data])

	if (!data.length && !globalFilter) {
		return null
	}

	return (
		<Card className="@container w-full px-3 py-5 sm:py-6 sm:px-6">
			<CardHeader className="p-0 mb-3 sm:mb-4">
				<div className="grid md:flex gap-x-5 gap-y-3 w-full items-end">
					<div className="px-2 sm:px-1">
						<CardTitle className="mb-2">
							<Trans>Processes</Trans>
						</CardTitle>
						<div className="text-sm text-muted-foreground flex items-center flex-wrap">
							<Trans>Total: {data.length}</Trans>
							<span className="text-primary/40 mx-2">|</span>
							<Trans>Running: {runningCount}</Trans>
						</div>
					</div>
					<Input
						placeholder={t`Filter...`}
						value={globalFilter}
						onChange={(e) => setGlobalFilter(e.target.value)}
						className="ms-auto px-4 w-full max-w-full md:w-64"
					/>
				</div>
			</CardHeader>
			<div className="rounded-md">
				<AllProcessTable table={table} rows={rows} colLength={visibleColumns.length} />
			</div>
		</Card>
	)
}

const AllProcessTable = memo(function AllProcessTable({
	table,
	rows,
	colLength,
}: {
	table: TableType<ProcessRecord>
	rows: Row<ProcessRecord>[]
	colLength: number
}) {
	const scrollRef = useRef<HTMLDivElement>(null)

	const virtualizer = useVirtualizer<HTMLDivElement, HTMLTableRowElement>({
		count: rows.length,
		estimateSize: () => 54,
		getScrollElement: () => scrollRef.current,
		overscan: 5,
	})
	const virtualRows = virtualizer.getVirtualItems()

	const paddingTop = Math.max(0, virtualRows[0]?.start ?? 0 - virtualizer.options.scrollMargin)
	const paddingBottom = Math.max(0, virtualizer.getTotalSize() - (virtualRows[virtualRows.length - 1]?.end ?? 0))

	return (
		<div
			className={cn(
				"h-min max-h-[calc(100dvh-17rem)] max-w-full relative overflow-auto border rounded-md",
				(!rows.length || rows.length > 2) && "min-h-50"
			)}
			ref={scrollRef}
		>
			<div style={{ height: `${virtualizer.getTotalSize() + 48}px`, paddingTop, paddingBottom }}>
				<table className="text-sm w-full h-full text-nowrap">
					<ProcessTableHead table={table} />
					<TableBody>
						{rows.length ? (
							virtualRows.map((virtualRow) => {
								const row = rows[virtualRow.index]
								return <ProcessTableRow key={row.id} row={row} virtualRow={virtualRow} />
							})
						) : (
							<TableRow>
								<TableCell colSpan={colLength} className="h-37 text-center pointer-events-none">
									<Trans>No results.</Trans>
								</TableCell>
							</TableRow>
						)}
					</TableBody>
				</table>
			</div>
		</div>
	)
})

function ProcessTableHead({ table }: { table: TableType<ProcessRecord> }) {
	return (
		<TableHeader className="sticky top-0 z-50 w-full border-b-2">
			{table.getHeaderGroups().map((headerGroup) => (
				<tr key={headerGroup.id}>
					{headerGroup.headers.map((header) => (
						<TableHead className="px-2" key={header.id}>
							{header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
						</TableHead>
					))}
				</tr>
			))}
		</TableHeader>
	)
}

const ProcessTableRow = memo(function ProcessTableRow({
	row,
	virtualRow,
}: {
	row: Row<ProcessRecord>
	virtualRow: VirtualItem
}) {
	return (
		<TableRow data-state={row.getIsSelected() && "selected"} className="transition-opacity">
			{row.getVisibleCells().map((cell) => (
				<TableCell key={cell.id} className="py-0" style={{ height: virtualRow.size }}>
					{flexRender(cell.column.columnDef.cell, cell.getContext())}
				</TableCell>
			))}
		</TableRow>
	)
})
