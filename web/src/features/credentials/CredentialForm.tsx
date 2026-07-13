import { BrowserQRCodeReader } from "@zxing/browser";
import { useForm } from "@tanstack/react-form";
import { FileScan, LoaderCircle, Sparkles } from "lucide-react";
import type { ChangeEvent } from "react";
import type { Algorithm, Credential, CredentialInput, CredentialSource, Group } from "@/api/types";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";

interface CredentialFormProps {
  credential: Credential | undefined;
  groups: Group[];
  onCancel: () => void;
  onSubmit: (input: CredentialInput) => Promise<void>;
}

export function CredentialForm({ credential, groups, onCancel, onSubmit }: CredentialFormProps) {
  const form = useForm({
    defaultValues: {
      source: "manual" as CredentialSource,
      issuer: credential?.issuer ?? "",
      account: credential?.account ?? "",
      secret: "",
      uri: "",
      algorithm: credential?.algorithm ?? ("SHA1" as Algorithm),
      digits: credential?.digits ?? 6,
      period: credential?.period ?? 30,
      favorite: credential?.favorite ?? false,
      groupId: credential?.groupId ?? "",
      tags: credential?.tags.join(", ") ?? "",
      notes: credential?.notes ?? "",
    },
    onSubmit: async ({ value }) => {
      const input: CredentialInput = {
        source: value.source,
        issuer: value.issuer,
        account: value.account,
        algorithm: value.algorithm,
        digits: value.digits,
        period: value.period,
        favorite: value.favorite,
        tags: value.tags
          .split(",")
          .map((tag) => tag.trim())
          .filter(Boolean),
        notes: value.notes,
        ...(value.secret === "" ? {} : { secret: value.secret }),
        ...(value.uri === "" ? {} : { uri: value.uri }),
        ...(value.groupId === "" ? {} : { groupId: value.groupId }),
      };
      await onSubmit(input);
    },
  });

  const scanQR = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file === undefined) return;
    const objectURL = URL.createObjectURL(file);
    try {
      const result = await new BrowserQRCodeReader().decodeFromImageUrl(objectURL);
      form.setFieldValue("uri", result.getText());
      form.setFieldValue("source", "uri");
    } finally {
      URL.revokeObjectURL(objectURL);
    }
  };

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onCancel();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <p className="text-primary text-xs font-bold tracking-[0.16em] uppercase">
            {credential === undefined ? "New test identity" : "Edit credential"}
          </p>
          <DialogTitle>
            {credential === undefined
              ? "Add TOTP credential"
              : `${credential.issuer}: ${credential.account}`}
          </DialogTitle>
          <DialogDescription>
            Seeds remain encrypted at rest and are only returned through explicit reveal.
          </DialogDescription>
        </DialogHeader>
        <form
          className="mt-6 grid gap-4 sm:grid-cols-2"
          onSubmit={(event) => {
            event.preventDefault();
            event.stopPropagation();
            void form.handleSubmit();
          }}
        >
          {credential === undefined && (
            <form.Field name="source">
              {(field) => (
                <Tabs
                  className="sm:col-span-2"
                  value={field.state.value}
                  onValueChange={(value) => field.handleChange(value as CredentialSource)}
                >
                  <TabsList className="grid w-full grid-cols-3">
                    <TabsTrigger value="manual">Manual</TabsTrigger>
                    <TabsTrigger value="uri">URI / QR</TabsTrigger>
                    <TabsTrigger value="generate">Generate</TabsTrigger>
                  </TabsList>
                </Tabs>
              )}
            </form.Field>
          )}
          <form.Subscribe selector={(state) => state.values.source}>
            {(source) =>
              source === "uri" ? (
                <div className="space-y-4 sm:col-span-2">
                  <form.Field
                    name="uri"
                    validators={{
                      onChange: ({ value }) =>
                        value.startsWith("otpauth://totp/")
                          ? undefined
                          : "Enter an otpauth://totp URI.",
                    }}
                  >
                    {(field) => (
                      <Field label="otpauth URI" error={field.state.meta.errors[0]}>
                        <Textarea
                          name={field.name}
                          value={field.state.value}
                          onBlur={field.handleBlur}
                          onChange={(event) => field.handleChange(event.target.value)}
                          placeholder="otpauth://totp/Example:developer…"
                        />
                      </Field>
                    )}
                  </form.Field>
                  <div className="rounded-lg border border-dashed p-4">
                    <Label htmlFor="qr-file" className="flex cursor-pointer items-center gap-2">
                      <FileScan className="text-primary size-4" />
                      Decode a QR image
                    </Label>
                    <Input
                      id="qr-file"
                      className="mt-3"
                      type="file"
                      accept="image/png,image/jpeg,image/webp,image/gif"
                      onChange={(event) => void scanQR(event)}
                    />
                    <p className="text-muted-foreground mt-2 text-xs">
                      Decoded in your browser; the image is never uploaded.
                    </p>
                  </div>
                </div>
              ) : (
                <>
                  <form.Field name="issuer">
                    {(field) => (
                      <Field label="Issuer">
                        <Input
                          name={field.name}
                          value={field.state.value}
                          onChange={(event) => field.handleChange(event.target.value)}
                          placeholder="Example app"
                        />
                      </Field>
                    )}
                  </form.Field>
                  <form.Field
                    name="account"
                    validators={{
                      onChange: ({ value }) =>
                        value.trim() === "" ? "Account is required." : undefined,
                    }}
                  >
                    {(field) => (
                      <Field label="Account" error={field.state.meta.errors[0]}>
                        <Input
                          name={field.name}
                          value={field.state.value}
                          onBlur={field.handleBlur}
                          onChange={(event) => field.handleChange(event.target.value)}
                          placeholder="developer@example.test"
                        />
                      </Field>
                    )}
                  </form.Field>
                  {source === "manual" && (
                    <form.Field
                      name="secret"
                      validators={{
                        onChange: ({ value }) =>
                          credential === undefined && value.trim() === ""
                            ? "Base32 seed is required."
                            : undefined,
                      }}
                    >
                      {(field) => (
                        <div className="sm:col-span-2">
                          <Field label="Base32 seed" error={field.state.meta.errors[0]}>
                            <Input
                              name={field.name}
                              value={field.state.value}
                              onBlur={field.handleBlur}
                              onChange={(event) => field.handleChange(event.target.value)}
                              placeholder={
                                credential === undefined
                                  ? "JBSWY3DPEHPK3PXP"
                                  : "Leave blank to keep the existing seed"
                              }
                              autoComplete="off"
                            />
                          </Field>
                        </div>
                      )}
                    </form.Field>
                  )}
                  <form.Field name="algorithm">
                    {(field) => (
                      <Field label="Algorithm">
                        <Select
                          value={field.state.value}
                          onValueChange={(value) => field.handleChange(value as Algorithm)}
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="SHA1">SHA-1</SelectItem>
                            <SelectItem value="SHA256">SHA-256</SelectItem>
                            <SelectItem value="SHA512">SHA-512</SelectItem>
                          </SelectContent>
                        </Select>
                      </Field>
                    )}
                  </form.Field>
                  <form.Field name="digits">
                    {(field) => (
                      <Field label="Digits">
                        <Select
                          value={String(field.state.value)}
                          onValueChange={(value) => field.handleChange(Number(value))}
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="6">6 digits</SelectItem>
                            <SelectItem value="7">7 digits</SelectItem>
                            <SelectItem value="8">8 digits</SelectItem>
                          </SelectContent>
                        </Select>
                      </Field>
                    )}
                  </form.Field>
                  <form.Field name="period">
                    {(field) => (
                      <Field label="Period (seconds)">
                        <Input
                          name={field.name}
                          type="number"
                          min={5}
                          max={300}
                          value={field.state.value}
                          onChange={(event) => field.handleChange(event.target.valueAsNumber)}
                        />
                      </Field>
                    )}
                  </form.Field>
                </>
              )
            }
          </form.Subscribe>
          <form.Field name="groupId">
            {(field) => (
              <Field label="Group">
                <Select
                  value={field.state.value === "" ? "none" : field.state.value}
                  onValueChange={(value) => field.handleChange(value === "none" ? "" : value)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">No group</SelectItem>
                    {groups.map((group) => (
                      <SelectItem key={group.id} value={group.id}>
                        {group.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
            )}
          </form.Field>
          <form.Field name="tags">
            {(field) => (
              <div className="sm:col-span-2">
                <Field label="Tags">
                  <Input
                    name={field.name}
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                    placeholder="staging, checkout, smoke-test"
                  />
                  <p className="text-muted-foreground mt-1 text-xs">Comma-separated</p>
                </Field>
              </div>
            )}
          </form.Field>
          <form.Field name="notes">
            {(field) => (
              <div className="sm:col-span-2">
                <Field label="Notes">
                  <Textarea
                    name={field.name}
                    maxLength={2000}
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                    placeholder="Purpose, test user, or environment context"
                  />
                </Field>
              </div>
            )}
          </form.Field>
          <form.Field name="favorite">
            {(field) => (
              <div className="flex items-center gap-3 sm:col-span-2">
                <Checkbox
                  id="favorite"
                  checked={field.state.value}
                  onCheckedChange={(value) => field.handleChange(value === true)}
                />
                <Label htmlFor="favorite">Pin this credential as a favorite</Label>
              </div>
            )}
          </form.Field>
          <DialogFooter className="sm:col-span-2">
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <form.Subscribe selector={(state) => [state.canSubmit, state.isSubmitting]}>
              {([canSubmit, submitting]) => (
                <Button type="submit" disabled={!canSubmit || submitting}>
                  {submitting ? <LoaderCircle className="animate-spin" /> : <Sparkles />}
                  {submitting
                    ? "Saving…"
                    : credential === undefined
                      ? "Add credential"
                      : "Save changes"}
                </Button>
              )}
            </form.Subscribe>
          </DialogFooter>
        </form>
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
