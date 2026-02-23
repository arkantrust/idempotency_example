/**
 * DeleteChargebackDialog asks for confirmation before deleting a chargeback.
 *
 * The DELETE endpoint is idempotent: deleting an already-deleted record returns
 * 200 OK, so retrying after a network failure is unconditionally safe.
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
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { deleteChargeback } from '@/lib/api'
import type { Chargeback } from '@/types/chargeback'

interface DeleteChargebackDialogProps {
  chargeback: Chargeback
}

export function DeleteChargebackDialog({ chargeback }: DeleteChargebackDialogProps) {
  const [open, setOpen] = useState(false)
  const qc = useQueryClient()

  const mutation = useMutation({
    mutationFn: () => deleteChargeback(chargeback.id),
    onSuccess: () => {
      toast.success(`Chargeback ${chargeback.id} deleted.`)
      qc.invalidateQueries({ queryKey: ['chargebacks'] })
      setOpen(false)
    },
    onError: (err: Error) => {
      toast.error(`Failed to delete: ${err.message}`)
    },
  })

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <Button variant="destructive" size="sm" onClick={() => setOpen(true)}>
        Delete
      </Button>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete Chargeback</DialogTitle>
          <DialogDescription>
            Are you sure you want to delete chargeback{' '}
            <code className="text-xs bg-slate-100 px-1 rounded">{chargeback.id}</code>?
            <br />
            This action is idempotent – retrying after a network failure is safe.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={() => mutation.mutate()}
            disabled={mutation.isPending}
          >
            {mutation.isPending ? 'Deleting…' : 'Delete'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
