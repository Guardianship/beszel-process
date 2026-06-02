import type { Column, ColumnDef } from "@tanstack/react-table"
import { Button } from "@/components/ui/button"
import { cn, hourWithSeconds } from "@/lib/utils"
import type { PortRecord } from "@/types"
import { ActivityIcon, ArrowUpDownIcon, ClockIcon, EthernetPortIcon, TerminalSquareIcon } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { t } from "@lingui/core/macro"

export const portTableCols: ColumnDef<PortRecord>[] = [
	{
		id: "service",
		sortingFn: (a, b) => a.original.service.localeCompare(b.original.service),
		accessorFn: (record) => record.service,
		header: ({ column }) => <HeaderButton column={column} name={t`Service`} Icon={EthernetPortIcon} />,
		cell: ({ getValue }) => {
			return <span className="ms-1.5 xl:w-50 block truncate">{getValue() as string}</span>
		},
	},
	{
		id: "port",
		accessorFn: (record) => record.port,
		header: ({ column }) => <HeaderButton column={column} name={t`Port`} Icon={EthernetPortIcon} />,
		cell: ({ getValue }) => {
			return <span className="ms-1.5 tabular-nums">{getValue() as number}</span>
		},
	},
	{
		id: "protocol",
		accessorFn: (record) => record.protocol,
		header: ({ column }) => <HeaderButton column={column} name={t`Protocol`} Icon={ActivityIcon} />,
		cell: ({ getValue }) => {
			return <span className="ms-1.5 uppercase">{getValue() as string}</span>
		},
	},
	{
		id: "status",
		accessorFn: (record) => record.status,
		header: ({ column }) => <HeaderButton column={column} name={t`Status`} Icon={ActivityIcon} />,
		cell: ({ getValue }) => {
			const status = getValue() as string
			const isOpen = status === "open"
			return (
				<Badge variant="outline" className="dark:border-white/12">
					<span className={cn("size-2 me-1.5 rounded-full", isOpen ? "bg-green-500" : "bg-red-500")} />
					{isOpen ? t`Open` : t`Closed`}
				</Badge>
			)
		},
	},
	{
		id: "process",
		sortingFn: (a, b) => (a.original.process || "").localeCompare(b.original.process || ""),
		accessorFn: (record) => record.process,
		header: ({ column }) => <HeaderButton column={column} name={t`Process`} Icon={TerminalSquareIcon} />,
		cell: ({ getValue }) => {
			const val = getValue() as string
			if (!val) return <span className="ms-1.5 text-muted-foreground">-</span>
			return <span className="ms-1.5 xl:w-40 block truncate">{val}</span>
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

function HeaderButton({ column, name, Icon }: { column: Column<PortRecord>; name: string; Icon: React.ElementType }) {
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
