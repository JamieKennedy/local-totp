import { useForm } from "@tanstack/react-form";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Copy, KeyRound, LoaderCircle, ShieldCheck } from "lucide-react";
import { useState } from "react";
import { api, APIError } from "@/api/client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dashboard } from "@/features/credentials/Dashboard";

export function SessionGate() {
  const queryClient = useQueryClient();
  const statusQuery = useQuery({ queryKey: ["status"], queryFn: api.status, retry: false });
  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey: ["status"] });
  };

  if (statusQuery.isPending) return <Loading />;
  if (statusQuery.isError) return <Loading error={messageOf(statusQuery.error)} />;
  if (!statusQuery.data.setup) return <SetupScreen onComplete={refresh} />;
  if (statusQuery.data.locked || !statusQuery.data.authenticated)
    return <UnlockScreen onComplete={refresh} />;
  return <Dashboard version={statusQuery.data.version} onLock={refresh} />;
}

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <main className="grid min-h-svh place-items-center bg-[radial-gradient(circle_at_20%_-10%,color-mix(in_oklab,var(--primary)_16%,transparent)_0,transparent_35rem)] p-4">
      <div className="w-full max-w-md">{children}</div>
    </main>
  );
}

function Loading({ error = "" }: { error?: string }) {
  return (
    <Shell>
      <div className="flex flex-col items-center gap-4 text-center">
        <div className="bg-primary text-primary-foreground grid size-14 place-items-center rounded-2xl font-black tracking-tighter">
          LT
        </div>
        <div>
          <h1 className="text-2xl font-bold">Local TOTP</h1>
          <p className="text-muted-foreground mt-1" role={error === "" ? undefined : "alert"}>
            {error === "" ? "Opening your local workbench…" : error}
          </p>
        </div>
        {error === "" && <LoaderCircle className="text-primary size-5 animate-spin" />}
      </div>
    </Shell>
  );
}

function SetupScreen({ onComplete }: { onComplete: () => Promise<void> }) {
  const [recovery, setRecovery] = useState("");
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");
  const form = useForm({
    defaultValues: { password: "", confirmation: "" },
    onSubmit: async ({ value }) => {
      if (value.password !== value.confirmation) {
        setError("Passwords do not match.");
        return;
      }
      try {
        setRecovery((await api.setup(value.password)).recoveryKey);
        setError("");
      } catch (caught) {
        setError(messageOf(caught));
      }
    },
  });

  if (recovery !== "") {
    return (
      <Shell>
        <Card>
          <CardHeader>
            <div className="bg-primary/15 text-primary mb-3 grid size-10 place-items-center rounded-lg">
              <ShieldCheck />
            </div>
            <CardTitle>Save your recovery key</CardTitle>
            <CardDescription>
              This is the only way to reset a forgotten master password. It will not be shown again.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            <code className="bg-muted block rounded-lg border p-4 text-sm leading-7 break-all">
              {recovery}
            </code>
            <Button
              variant="outline"
              className="w-full"
              onClick={() => void navigator.clipboard.writeText(recovery)}
            >
              <Copy />
              Copy recovery key
            </Button>
            <div className="flex items-start gap-3">
              <Checkbox
                id="saved"
                checked={saved}
                onCheckedChange={(value) => setSaved(value === true)}
              />
              <Label htmlFor="saved" className="text-muted-foreground leading-5">
                I saved this recovery key outside the vault.
              </Label>
            </div>
            <Button className="w-full" disabled={!saved} onClick={() => void onComplete()}>
              <Check />
              Open workbench
            </Button>
          </CardContent>
        </Card>
      </Shell>
    );
  }

  return (
    <Shell>
      <Card>
        <CardHeader>
          <p className="text-primary text-xs font-bold tracking-[0.18em] uppercase">
            Test and staging only
          </p>
          <CardTitle>Create your local vault</CardTitle>
          <CardDescription>
            Keep development credentials out of your personal authenticator. Never add production or
            personal MFA seeds.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form
            className="space-y-4"
            onSubmit={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void form.handleSubmit();
            }}
          >
            <form.Field
              name="password"
              validators={{
                onChange: ({ value }) =>
                  value.length >= 12 ? undefined : "Use at least 12 characters.",
              }}
            >
              {(field) => (
                <Field label="Master password" error={field.state.meta.errors[0]}>
                  <Input
                    name={field.name}
                    type="password"
                    autoComplete="new-password"
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                </Field>
              )}
            </form.Field>
            <form.Field name="confirmation">
              {(field) => (
                <Field label="Confirm password">
                  <Input
                    name={field.name}
                    type="password"
                    autoComplete="new-password"
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                </Field>
              )}
            </form.Field>
            {error !== "" && (
              <p className="text-destructive text-sm" role="alert">
                {error}
              </p>
            )}
            <form.Subscribe selector={(state) => [state.canSubmit, state.isSubmitting]}>
              {([canSubmit, isSubmitting]) => (
                <Button className="w-full" type="submit" disabled={!canSubmit || isSubmitting}>
                  {isSubmitting ? <LoaderCircle className="animate-spin" /> : <KeyRound />}
                  {isSubmitting ? "Creating vault…" : "Create encrypted vault"}
                </Button>
              )}
            </form.Subscribe>
          </form>
        </CardContent>
      </Card>
    </Shell>
  );
}

function UnlockScreen({ onComplete }: { onComplete: () => Promise<void> }) {
  const [recoveryMode, setRecoveryMode] = useState(false);
  const [rotatedRecovery, setRotatedRecovery] = useState("");
  const [error, setError] = useState("");
  const form = useForm({
    defaultValues: { password: "", recoveryKey: "", replacement: "" },
    onSubmit: async ({ value }) => {
      try {
        if (recoveryMode)
          setRotatedRecovery(await api.recover(value.recoveryKey, value.replacement));
        else {
          await api.unlock(value.password);
          await onComplete();
        }
        setError("");
      } catch (caught) {
        setError(messageOf(caught));
      }
    },
  });

  if (rotatedRecovery !== "") {
    return (
      <Shell>
        <Card>
          <CardHeader>
            <CardTitle>Save your new recovery key</CardTitle>
            <CardDescription>The previous key has been revoked.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <code className="bg-muted block rounded-lg border p-4 text-sm leading-7 break-all">
              {rotatedRecovery}
            </code>
            <Button
              variant="outline"
              className="w-full"
              onClick={() => void navigator.clipboard.writeText(rotatedRecovery)}
            >
              <Copy />
              Copy key
            </Button>
            <Button className="w-full" onClick={() => void onComplete()}>
              Open workbench
            </Button>
          </CardContent>
        </Card>
      </Shell>
    );
  }

  return (
    <Shell>
      <Card>
        <CardHeader>
          <p className="text-primary text-xs font-bold tracking-[0.18em] uppercase">
            Local encrypted vault
          </p>
          <CardTitle>{recoveryMode ? "Recover access" : "Unlock Local TOTP"}</CardTitle>
          <CardDescription>
            {recoveryMode
              ? "Recovery rotates your password and recovery key."
              : "Your vault remains unlocked until you lock it or stop the server."}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form
            className="space-y-4"
            onSubmit={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void form.handleSubmit();
            }}
          >
            {recoveryMode ? (
              <>
                <form.Field name="recoveryKey">
                  {(field) => (
                    <Field label="Recovery key">
                      <Input
                        name={field.name}
                        value={field.state.value}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                    </Field>
                  )}
                </form.Field>
                <form.Field
                  name="replacement"
                  validators={{
                    onChange: ({ value }) =>
                      value.length >= 12 ? undefined : "Use at least 12 characters.",
                  }}
                >
                  {(field) => (
                    <Field label="New master password" error={field.state.meta.errors[0]}>
                      <Input
                        name={field.name}
                        type="password"
                        value={field.state.value}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                    </Field>
                  )}
                </form.Field>
              </>
            ) : (
              <form.Field name="password">
                {(field) => (
                  <Field label="Master password">
                    <Input
                      name={field.name}
                      type="password"
                      autoComplete="current-password"
                      value={field.state.value}
                      onChange={(event) => field.handleChange(event.target.value)}
                    />
                  </Field>
                )}
              </form.Field>
            )}
            {error !== "" && (
              <p className="text-destructive text-sm" role="alert">
                {error}
              </p>
            )}
            <form.Subscribe selector={(state) => state.isSubmitting}>
              {(isSubmitting) => (
                <Button className="w-full" type="submit" disabled={isSubmitting}>
                  {isSubmitting && <LoaderCircle className="animate-spin" />}
                  {recoveryMode ? "Recover and rotate key" : "Unlock"}
                </Button>
              )}
            </form.Subscribe>
          </form>
          <Button
            variant="ghost"
            className="text-muted-foreground mt-2 w-full"
            onClick={() => {
              setRecoveryMode(!recoveryMode);
              setError("");
            }}
          >
            {recoveryMode ? "Back to password" : "I forgot my password"}
          </Button>
        </CardContent>
      </Card>
    </Shell>
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

function messageOf(value: unknown): string {
  if (value instanceof APIError || value instanceof Error) return value.message;
  return "Something went wrong.";
}
