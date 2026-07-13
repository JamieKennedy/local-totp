import { useForm } from "@tanstack/react-form";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Copy, Download, Key, Plus, RefreshCw, Save, Trash2, Upload } from "lucide-react";
import { useState } from "react";
import { api } from "@/api/client";
import type { BackupPreview, CreatedAPIKey } from "@/api/types";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export function Settings({ onDataChange }: { onDataChange: () => Promise<void> }) {
  const queryClient = useQueryClient();
  const groupsQuery = useQuery({ queryKey: ["groups"], queryFn: api.groups });
  const apiKeysQuery = useQuery({ queryKey: ["apiKeys"], queryFn: api.apiKeys });
  const [createdKey, setCreatedKey] = useState<CreatedAPIKey>();
  const [recovery, setRecovery] = useState("");
  const [backupFile, setBackupFile] = useState<File>();
  const [backupPreview, setBackupPreview] = useState<BackupPreview>();
  const [message, setMessage] = useState("");

  const groupForm = useForm({
    defaultValues: { name: "", color: "#14b8a6" },
    onSubmit: async ({ value, formApi }) => {
      await api.createGroup(value.name, value.color);
      formApi.reset();
      await Promise.all([queryClient.invalidateQueries({ queryKey: ["groups"] }), onDataChange()]);
    },
  });
  const keyForm = useForm({
    defaultValues: { name: "" },
    onSubmit: async ({ value, formApi }) => {
      setCreatedKey(await api.createAPIKey(value.name));
      formApi.reset();
      await queryClient.invalidateQueries({ queryKey: ["apiKeys"] });
    },
  });
  const passwordForm = useForm({
    defaultValues: { current: "", replacement: "", confirmation: "" },
    onSubmit: async ({ value, formApi }) => {
      if (value.replacement !== value.confirmation) {
        setMessage("New passwords do not match.");
        return;
      }
      await api.changePassword(value.current, value.replacement);
      formApi.reset();
      setMessage("Master password changed.");
    },
  });
  const backupForm = useForm({
    defaultValues: { password: "" },
    onSubmit: async ({ value }) => {
      if (backupFile === undefined) return;
      setBackupPreview(await api.previewBackup(backupFile, value.password));
    },
  });
  const exportForm = useForm({
    defaultValues: { password: "" },
    onSubmit: async ({ value, formApi }) => {
      const blob = await api.exportBackup(value.password);
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `local-totp-${new Date().toISOString().slice(0, 10)}.ltotp`;
      link.click();
      URL.revokeObjectURL(url);
      formApi.reset();
    },
  });
  const deleteGroup = useMutation({
    mutationFn: api.deleteGroup,
    onSuccess: async () => {
      await Promise.all([queryClient.invalidateQueries({ queryKey: ["groups"] }), onDataChange()]);
    },
  });
  const deleteKey = useMutation({
    mutationFn: api.deleteAPIKey,
    onSuccess: async () => queryClient.invalidateQueries({ queryKey: ["apiKeys"] }),
  });
  const applyBackup = useMutation({
    mutationFn: (mode: "merge" | "replace") =>
      backupPreview === undefined ? Promise.resolve() : api.applyBackup(backupPreview.id, mode),
    onSuccess: async () => {
      setBackupPreview(undefined);
      setBackupFile(undefined);
      await Promise.all([queryClient.invalidateQueries(), onDataChange()]);
    },
  });

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle>Groups</CardTitle>
          <CardDescription>Each credential can belong to one folder-like group.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <form
            className="flex gap-2"
            onSubmit={(event) => {
              event.preventDefault();
              void groupForm.handleSubmit();
            }}
          >
            <groupForm.Field
              name="name"
              validators={{
                onChange: ({ value }) => (value.trim() === "" ? "Required" : undefined),
              }}
            >
              {(field) => (
                <Input
                  aria-label="Group name"
                  placeholder="Staging"
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                />
              )}
            </groupForm.Field>
            <groupForm.Field name="color">
              {(field) => (
                <Input
                  aria-label="Group color"
                  className="w-14 p-1"
                  type="color"
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                />
              )}
            </groupForm.Field>
            <Button size="icon" type="submit" aria-label="Create group">
              <Plus />
            </Button>
          </form>
          <div className="divide-y rounded-lg border">
            {(groupsQuery.data ?? []).length === 0 ? (
              <p className="text-muted-foreground p-4 text-sm">No groups yet.</p>
            ) : (
              groupsQuery.data?.map((group) => (
                <div key={group.id} className="flex items-center justify-between p-3">
                  <div className="flex items-center gap-3">
                    <span
                      className="size-3 rounded-full"
                      style={{ backgroundColor: group.color }}
                    />
                    <span className="font-medium">{group.name}</span>
                  </div>
                  <Button
                    size="icon"
                    variant="ghost"
                    className="text-destructive"
                    onClick={() => deleteGroup.mutate(group.id)}
                  >
                    <Trash2 />
                  </Button>
                </div>
              ))
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Read-only API keys</CardTitle>
          <CardDescription>
            Use named keys from scripts or the Local TOTP CLI. Keys cannot reveal seeds or modify
            data.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <form
            className="flex gap-2"
            onSubmit={(event) => {
              event.preventDefault();
              void keyForm.handleSubmit();
            }}
          >
            <keyForm.Field
              name="name"
              validators={{
                onChange: ({ value }) => (value.trim() === "" ? "Required" : undefined),
              }}
            >
              {(field) => (
                <Input
                  aria-label="API key name"
                  placeholder="Local smoke tests"
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                />
              )}
            </keyForm.Field>
            <Button type="submit">
              <Key />
              Create key
            </Button>
          </form>
          <div className="divide-y rounded-lg border">
            {(apiKeysQuery.data ?? []).length === 0 ? (
              <p className="text-muted-foreground p-4 text-sm">No API keys.</p>
            ) : (
              apiKeysQuery.data?.map((key) => (
                <div key={key.id} className="flex items-center justify-between p-3">
                  <div>
                    <p className="font-medium">{key.name}</p>
                    <p className="text-muted-foreground text-xs">
                      Created {new Date(key.createdAt).toLocaleDateString()}
                      {key.lastUsed === undefined
                        ? " · Never used"
                        : ` · Used ${new Date(key.lastUsed).toLocaleString()}`}
                    </p>
                  </div>
                  <Button
                    size="icon"
                    variant="ghost"
                    className="text-destructive"
                    onClick={() => deleteKey.mutate(key.id)}
                  >
                    <Trash2 />
                  </Button>
                </div>
              ))
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Encrypted backups</CardTitle>
          <CardDescription>
            Exports use the current master password with a fresh Argon2id salt.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-6 sm:grid-cols-2">
          <form
            className="space-y-3"
            onSubmit={(event) => {
              event.preventDefault();
              void exportForm.handleSubmit();
            }}
          >
            <h3 className="font-semibold">Export</h3>
            <exportForm.Field name="password">
              {(field) => (
                <Field label="Master password">
                  <Input
                    type="password"
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                </Field>
              )}
            </exportForm.Field>
            <Button variant="outline" className="w-full" type="submit">
              <Download />
              Download .ltotp
            </Button>
          </form>
          <form
            className="space-y-3"
            onSubmit={(event) => {
              event.preventDefault();
              void backupForm.handleSubmit();
            }}
          >
            <h3 className="font-semibold">Import</h3>
            <Input
              aria-label="Backup file"
              type="file"
              accept=".ltotp,application/json"
              onChange={(event) => setBackupFile(event.target.files?.[0])}
            />
            <backupForm.Field name="password">
              {(field) => (
                <Field label="Backup password">
                  <Input
                    type="password"
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                </Field>
              )}
            </backupForm.Field>
            <Button
              variant="outline"
              className="w-full"
              type="submit"
              disabled={backupFile === undefined}
            >
              <Upload />
              Preview import
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Password and recovery</CardTitle>
          <CardDescription>
            Changing the password rewraps the vault key; it does not rewrite every record.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <form
            className="grid gap-3 sm:grid-cols-2"
            onSubmit={(event) => {
              event.preventDefault();
              void passwordForm.handleSubmit();
            }}
          >
            <passwordForm.Field name="current">
              {(field) => (
                <div className="sm:col-span-2">
                  <Field label="Current password">
                    <Input
                      type="password"
                      value={field.state.value}
                      onChange={(event) => field.handleChange(event.target.value)}
                    />
                  </Field>
                </div>
              )}
            </passwordForm.Field>
            <passwordForm.Field
              name="replacement"
              validators={{
                onChange: ({ value }) =>
                  value.length >= 12 ? undefined : "Use 12 or more characters.",
              }}
            >
              {(field) => (
                <Field label="New password" error={field.state.meta.errors[0]}>
                  <Input
                    type="password"
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                </Field>
              )}
            </passwordForm.Field>
            <passwordForm.Field name="confirmation">
              {(field) => (
                <Field label="Confirm password">
                  <Input
                    type="password"
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                </Field>
              )}
            </passwordForm.Field>
            <Button className="sm:col-span-2" type="submit">
              <Save />
              Change password
            </Button>
          </form>
          {message !== "" && (
            <p className="text-muted-foreground text-sm" role="status">
              {message}
            </p>
          )}
          <div className="border-t pt-4">
            <Button
              variant="outline"
              className="w-full"
              onClick={() => void api.rotateRecovery().then(setRecovery)}
            >
              <RefreshCw />
              Rotate recovery key
            </Button>
          </div>
        </CardContent>
      </Card>

      {createdKey !== undefined && (
        <OneTimeValue
          title="Copy this API key now"
          description="Only its hash is stored. Put it in a protected file for the CLI."
          value={createdKey.key}
          onClose={() => setCreatedKey(undefined)}
        />
      )}
      {recovery !== "" && (
        <OneTimeValue
          title="Save the new recovery key"
          description="The previous recovery key was revoked immediately."
          value={recovery}
          onClose={() => setRecovery("")}
        />
      )}
      {backupPreview !== undefined && (
        <Dialog
          open
          onOpenChange={(open) => {
            if (!open) setBackupPreview(undefined);
          }}
        >
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Backup import preview</DialogTitle>
              <DialogDescription>
                No data changes until you choose merge or replace.
              </DialogDescription>
            </DialogHeader>
            <div className="grid grid-cols-2 gap-3 py-4 sm:grid-cols-4">
              <Stat label="Credentials" value={backupPreview.credentials} />
              <Stat label="Groups" value={backupPreview.groups} />
              <Stat label="Duplicates" value={backupPreview.duplicates} />
              <Stat label="Name conflicts" value={backupPreview.nameConflicts} />
            </div>
            <div className="border-destructive/30 bg-destructive/5 text-muted-foreground rounded-lg border p-3 text-sm">
              <strong className="text-foreground">Replace</strong> removes current credentials and
              groups. API keys and the current vault identity remain.
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => applyBackup.mutate("merge")}>
                <Upload />
                Merge
              </Button>
              <Button variant="destructive" onClick={() => applyBackup.mutate("replace")}>
                <Trash2 />
                Replace current data
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}

function OneTimeValue({
  title,
  description,
  value,
  onClose,
}: {
  title: string;
  description: string;
  value: string;
  onClose: () => void;
}) {
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <code className="bg-muted my-4 block rounded-lg border p-4 text-sm leading-7 break-all">
          {value}
        </code>
        <DialogFooter>
          <Button variant="outline" onClick={() => void navigator.clipboard.writeText(value)}>
            <Copy />
            Copy
          </Button>
          <Button onClick={onClose}>I saved it</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function Field({
  label,
  error,
  children,
}: {
  label: string;
  error?: unknown;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      {children}
      {typeof error === "string" && <p className="text-destructive text-xs">{error}</p>}
    </div>
  );
}
function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div className="bg-muted/40 rounded-lg border p-3 text-center">
      <p className="text-2xl font-bold">{value}</p>
      <p className="text-muted-foreground text-xs">{label}</p>
    </div>
  );
}
