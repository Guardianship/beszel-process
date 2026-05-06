import type { Column, ColumnDef } from "@tanstack/react-table"
import { Button } from "@/components/ui/button"
import { cn, decimalString, formatBytes, hourWithSeconds } from "@/lib/utils"
import type { ProcessRecord } from "@/types"
import { ActivityIcon, ArrowUpDownIcon, ClockIcon, CpuIcon, MemoryStickIcon, TerminalSquareIcon } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { t } from "@lingui/core/macro"

export const processTableCols: ColumnDef<ProcessRecord>[] = [
	{
		id: "name",
		sortingFn: (a, b) => a.original.name.localeCompare(b.original.name),
		accessorFn: (record) => record.name,
		header: ({ column }) => <HeaderButton column={column} name={t`Name`} Icon={TerminalSquareIcon} />,
		cell: ({ getValue }) => {
			return <span className="ms-1.5 xl:w-50 block truncate">{getValue() as string}</span>
		},
	},
	{
		id: "pid",
		accessorFn: (record) => record.pid,
		header: ({ column }) => <HeaderButton column={column} name={t`PID`} Icon={ActivityIcon} />,
		cell: ({ getValue }) => {
			const val = getValue() as number
			if (!val) return <span className="ms-1.5 text-muted-foreground">-</span>
			return <span className="ms-1.5 tabular-nums">{val}</span>
		},
	},
	{
		id: "status",
		accessorFn: (record) => record.status,
		header: ({ column }) => <HeaderButton column={column} name={t`Status`} Icon={ActivityIcon} />,
		cell: ({ getValue }) => {
			const status = getValue() as string
			const isRunning = status !== "stopped"
			return (
				<Badge variant="outline" className="dark:border-white/12">
					<span className={cn("size-2 me-1.5 rounded-full", isRunning ? "bg-green-500" : "bg-red-500")} />
					{isRunning ? t`Running` : t`Stopped`}
				</Badge>
			)
		},
	},
	{
		id: "cpu",
		accessorFn: (record) => record.cpu,
		invertSorting: true,
		header: ({ column }) => <HeaderButton column={column} name={t`CPU`} Icon={CpuIcon} />,
		cell: ({ getValue }) => {
			const val = getValue() as number
			if (!val) return <span className="ms-1.5 text-muted-foreground">-</span>
			return <span className="ms-1.5 tabular-nums">{`${decimalString(val, val >= 10 ? 1 : 2)}%`}</span>
		},
	},
	{
		id: "memory",
		accessorFn: (record) => record.memory,
		invertSorting: true,
		header: ({ column }) => <HeaderButton column={column} name={t`Memory`} Icon={MemoryStickIcon} />,
		cell: ({ getValue }) => {
			const val = getValue() as number
			if (!val) return <span className="ms-1.5 text-muted-foreground">-</span>
			const formatted = formatBytes(val * 1024 * 1024, false, undefined, false)
			return (
				<span className="ms-1.5 tabular-nums">{`${decimalString(formatted.value, formatted.value >= 10 ? 1 : 2)} ${formatted.unit}`}</span>
			)
		},
	},
	{
		id: "uptime",
		accessorFn: (record) => record.uptime,
		invertSorting: true,
		header: ({ column }) => <HeaderButton column={column} name={t`Uptime`} Icon={ClockIcon} />,
		cell: ({ getValue }) => {
			const seconds = getValue() as number
			if (!seconds) return <span className="ms-1.5 text-muted-foreground">-</span>
			const days = Math.floor(seconds / 86400)
			const hours = Math.floor((seconds % 86400) / 3600)
			const mins = Math.floor((seconds % 3600) / 60)
			if (days > 0) return <span className="ms-1.5 tabular-nums">{`${days}d ${hours}h`}</span>
			if (hours > 0) return <span className="ms-1.5 tabular-nums">{`${hours}h ${mins}m`}</span>
			return <span className="ms-1.5 tabular-nums">{`${mins}m`}</span>
		},
	},
	{
		id: "updated",
		invertSorting: true,
		accessorFn: (record) => record.updated,
		header: ({ column }) => <HeaderButton column={column} name={t`Updated`} Icon={ClockIcon} />,
		cell: ({ getValue }) => {
			const timestamp = getValue() as string
			if (!timestamp) return null
			return <span className="ms-1.5 tabular-nums">{hourWithSeconds(new Date(timestamp).toISOString())}</span>
		},
	},
]

function HeaderButton({
	column,
	name,
	Icon,
}: {
	column: Column<ProcessRecord>
	name: string
	Icon: React.ElementType
}) {
	const isSorted = column.getIsSorted()
	return (
		<Button
			className={cn(
				"h-9 px-3 flex items-center gap-2 duration-50",
				isSorted && "bg-accent/70 light:bg-accent text-accent-foreground/90"
			)}
			variant="ghost"
			onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
		>
			{Icon && <Icon className="size-4" />}
			{name}
			<ArrowUpDownIcon className="size-4" />
		</Button>
	)
}
