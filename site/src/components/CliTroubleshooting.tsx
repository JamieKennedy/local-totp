import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";

const issues = [
  {
    question: "The CLI says “no API key configured”",
    answer:
      "Create a named API key in the web interface, store it in a user-readable file, and run configure with --api-key-file. You can instead set LOCAL_TOTP_API_KEY_FILE for the current environment.",
  },
  {
    question: "Authentication fails with “unauthorized”",
    answer:
      "Confirm the key file contains exactly the one-time plaintext API key, with no surrounding quotes. If the key was revoked in Settings, create and configure a replacement.",
  },
  {
    question: "A credential name is ambiguous",
    answer:
      "The code command matches issuer:account case-insensitively. When multiple records share the same display name, use one of the UUIDs listed in the error or returned by local-totp list.",
  },
  {
    question: "The vault is locked",
    answer:
      "Unlock it in the local web interface. API keys remain read-only credentials and cannot unlock, configure, or modify the vault.",
  },
  {
    question: "The CLI cannot connect",
    answer:
      "Run local-totp status, verify the configured URL, and check http://localhost:8080/healthz. Local TOTP accepts only loopback Host values, so do not configure a remote hostname.",
  },
];

export function CliTroubleshooting() {
  return (
    <Accordion
      type="single"
      collapsible
      className="not-prose my-6 rounded-xl border px-5"
    >
      {issues.map((issue, index) => (
        <AccordionItem key={issue.question} value={`issue-${String(index)}`}>
          <AccordionTrigger>{issue.question}</AccordionTrigger>
          <AccordionContent>{issue.answer}</AccordionContent>
        </AccordionItem>
      ))}
    </Accordion>
  );
}
