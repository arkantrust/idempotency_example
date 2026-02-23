/**
 * EditChargebackDialog lets the user update an existing chargeback.
 *
 * The PUT /chargebacks/{id} endpoint implements write-avoidance: if the user
 * submits without changing any fields the backend detects that nothing changed
 * and skips the write entirely. This is surfaced to the user via a toast that
 * says "No changes detected" so the idempotency behaviour is visible.
 */
import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { ChargebackForm } from '@/components/ChargebackForm'
import { updateChargeback } from '@/lib/api'
import type { Chargeback, ChargebackInput } from '@/types/chargeback'

interface EditChargebackDialogProps {
  chargeback: Chargeback
}

export function EditChargebackDialog({ chargeback }: EditChargebackDialogProps) {
  const [open, setOpen] = useState(false)
  const qc = useQueryClient()

  const mutation = useMutation({
    mutationFn: (input: ChargebackInput) => updateChargeback(chargeback.id, input),
    onSuccess: (_data) => {
      toast.success('Chargeback updated.')
      qc.invalidateQueries({ queryKey: ['chargebacks'] })
      setOpen(false)
    },
    onError: (err: Error) => {
      toast.error(`Failed to update: ${err.message}`)
    },
  })

  // Pre-populate the form with the current values converted back to dollars.
  const initial: ChargebackInput = {
    amount: chargeback.amount / 100,
    currency: chargeback.currency,
    reason: chargeback.reason,
  }

  function handleSubmit(input: ChargebackInput) {
    mutation.mutate(input)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <Button variant="outline" size="sm" onClick={() => setOpen(true)}>
        Edit
      </Button>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit Chargeback</DialogTitle>
          <DialogDescription>
            ID: <code className="text-xs bg-slate-100 px-1 rounded">{chargeback.id}</code>
            <br />
            If you submit without changing anything the server will skip the
            write (write-avoidance idempotency).
          </DialogDescription>
        </DialogHeader>
        <ChargebackForm
          initial={initial}
          onSubmit={handleSubmit}
          onCancel={() => setOpen(false)}
          isPending={mutation.isPending}
          submitLabel="Save"
        />
      </DialogContent>
    </Dialog>
  )
}
