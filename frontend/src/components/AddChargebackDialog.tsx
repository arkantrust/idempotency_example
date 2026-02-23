/**
 * AddChargebackDialog lets the user create a new chargeback.
 *
 * A UUID is generated client-side and used as the idempotency key for the
 * POST /chargebacks/{id} request. If the dialog is submitted, closed, and then
 * re-opened with the same form data, re-submitting would use a new UUID –
 * preventing accidental duplicate submissions while still demonstrating that
 * retrying with the SAME UUID is safe (the backend returns the existing record
 * instead of creating a duplicate).
 */
import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { ChargebackForm } from '@/components/ChargebackForm'
import { createChargeback } from '@/lib/api'
import type { ChargebackInput } from '@/types/chargeback'

// Minimal UUID v4 generator (crypto.randomUUID is available in modern browsers).
function newId(): string {
  return crypto.randomUUID()
}

export function AddChargebackDialog() {
  const [open, setOpen] = useState(false)
  const qc = useQueryClient()

  const mutation = useMutation({
    mutationFn: ({ id, input }: { id: string; input: ChargebackInput }) =>
      createChargeback(id, input),
    onSuccess: () => {
      toast.success('Chargeback created successfully.')
      qc.invalidateQueries({ queryKey: ['chargebacks'] })
      setOpen(false)
    },
    onError: (err: Error) => {
      toast.error(`Failed to create chargeback: ${err.message}`)
    },
  })

  function handleSubmit(input: ChargebackInput) {
    mutation.mutate({ id: newId(), input })
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>+ Add Chargeback</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Chargeback</DialogTitle>
          <DialogDescription>
            A UUID will be generated as the idempotency key. Retrying the same
            request is safe – the server returns the existing record.
          </DialogDescription>
        </DialogHeader>
        <ChargebackForm
          onSubmit={handleSubmit}
          onCancel={() => setOpen(false)}
          isPending={mutation.isPending}
          submitLabel="Create"
        />
      </DialogContent>
    </Dialog>
  )
}
