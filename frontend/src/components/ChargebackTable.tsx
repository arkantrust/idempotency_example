/**
 * ChargebackTable renders the main data table listing all chargebacks.
 *
 * It uses React Query's useQuery to fetch data and shows loading/error states.
 * Each row has Edit and Delete buttons that open their respective dialogs.
 */
import { useQuery } from '@tanstack/react-query'
import { listChargebacks } from '@/lib/api'
import { EditChargebackDialog } from '@/components/EditChargebackDialog'
import { DeleteChargebackDialog } from '@/components/DeleteChargebackDialog'

export function ChargebackTable() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['chargebacks'],
    queryFn: listChargebacks,
  })

  if (isLoading) {
    return <p className="text-slate-500 text-sm py-8 text-center">Loading chargebacks…</p>
  }

  if (isError) {
    return (
      <p className="text-red-600 text-sm py-8 text-center">
        Failed to load chargebacks: {(error as Error).message}
      </p>
    )
  }

  if (!data || data.length === 0) {
    return (
      <p className="text-slate-400 text-sm py-8 text-center">
        No chargebacks yet. Click <strong>+ Add Chargeback</strong> to create one.
      </p>
    )
  }

  return (
    <div className="overflow-x-auto rounded-lg border border-slate-200">
      <table className="w-full text-sm">
        <thead className="bg-slate-50 text-slate-600 uppercase text-xs">
          <tr>
            <th className="px-4 py-3 text-left font-medium">ID</th>
            <th className="px-4 py-3 text-right font-medium">Amount</th>
            <th className="px-4 py-3 text-left font-medium">Currency</th>
            <th className="px-4 py-3 text-left font-medium">Reason</th>
            <th className="px-4 py-3 text-left font-medium">Created</th>
            <th className="px-4 py-3 text-left font-medium">Updated</th>
            <th className="px-4 py-3 text-left font-medium">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {data.map((cb) => (
            <tr key={cb.id} className="hover:bg-slate-50 transition-colors">
              <td className="px-4 py-3 font-mono text-xs text-slate-500 max-w-[160px] truncate">
                {cb.id}
              </td>
              <td className="px-4 py-3 text-right tabular-nums">
                {(cb.amount / 100).toFixed(2)}
              </td>
              <td className="px-4 py-3">{cb.currency}</td>
              <td className="px-4 py-3 max-w-[200px] truncate">{cb.reason}</td>
              <td className="px-4 py-3 text-slate-500 text-xs whitespace-nowrap">
                {new Date(cb.createdAt).toLocaleString()}
              </td>
              <td className="px-4 py-3 text-slate-500 text-xs whitespace-nowrap">
                {new Date(cb.updatedAt).toLocaleString()}
              </td>
              <td className="px-4 py-3">
                <div className="flex gap-2">
                  <EditChargebackDialog chargeback={cb} />
                  <DeleteChargebackDialog chargeback={cb} />
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
