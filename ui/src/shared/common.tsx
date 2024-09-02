import { Secret } from "./models";

export function splitSecret(secret: string) : Secret {
  if (secret == null) {
      return new Secret();
  }
  const output = new Secret();
  const secretParts = secret.split(" ");
  output.name = secretParts[0];
  output.namespace = secretParts[2].replace("(", "");
  output.namespace = output.namespace.replace(")", "");
  return output;
}

export function generateRandom5CharString() : string {
  return Math.random().toString(20).substring(2, 7)
}
