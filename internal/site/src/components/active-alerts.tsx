import { alertInfo } from "@/lib/alerts"
import { $alerts, $allSystemsById } from "@/lib/stores"
import type { AlertRecord } from "@/types"
import { Trans } from "@lingui/react/macro"
import { useStore } from "@nanostores/react"
import { getPagePath } from "@nanostores/router"
import { useMemo } from "react"
import { $router, Link } from "./router"
import { Alert, AlertTitle, AlertDescription } from "./ui/alert"
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card"

export const ActiveAlerts = () => {
	const alerts = useStore($alerts)
	const systems = useStore($allSystemsById)

	const { allAlerts, allAlertsKey } = useMemo(() => {
		const allAlerts: AlertRecord[] = []
		const allAlertsKey: string[] = []

		for (const systemId of Object.keys(alerts)) {
			for (const alert of alerts[systemId].values()) {
				if (alert.name in alertInfo) {
					allAlerts.push(alert)
					allAlertsKey.push(`${alert.system}${alert.value}${alert.min}${alert.item}${alert.triggered}`)
				}
			}
		}

		return { allAlerts, allAlertsKey }
	}, [alerts])

	// biome-ignore lint/correctness/useExhaustiveDependencies: allAlertsKey is inclusive
	return useMemo(() => {
		if (allAlerts.length === 0) {
			return null
		}
		return (
			<Card>
				<CardHeader className="pb-4 px-2 sm:px-6 max-sm:pt-5 max-sm:pb-1">
					<div className="px-2 sm:px-1">
						<CardTitle>
							<Trans>All Configured Alerts</Trans>
						</CardTitle>
					</div>
				</CardHeader>
				<CardContent className="max-sm:p-2">
					<div className="grid sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4 gap-3">
						{allAlerts.map((alert) => {
							const info = alertInfo[alert.name as keyof typeof alertInfo]
							const isTriggered = alert.triggered
							return (
								<Alert
									key={alert.id}
									className={`hover:-translate-y-px duration-200 bg-transparent border-foreground/10 hover:shadow-md shadow-black/5 ${
										isTriggered ? "border-red-500/50" : "border-green-500/50"
									}`}
								>
									<info.icon className="h-4 w-4" />
									<AlertTitle>
										{systems[alert.system]?.name} {info.name()}
										{alert.item ? `: ${alert.item}` : ""}
									</AlertTitle>
									<AlertDescription>
										{isTriggered ? (
											<span className="text-red-500">
												<Trans>Triggered</Trans>
											</span>
										) : (
											<span className="text-green-500">
												<Trans>Enabled</Trans>
											</span>
										)}
									</AlertDescription>
									<Link
										href={getPagePath($router, "system", { id: systems[alert.system]?.id })}
										className="absolute inset-0 w-full h-full"
										aria-label="View system"
									></Link>
								</Alert>
							)
						})}
					</div>
				</CardContent>
			</Card>
		)
	}, [allAlertsKey.join("")])
}
