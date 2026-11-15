import { readFile } from "node:fs/promises";

export async function getDialogs() {
  const filePath = new URL("./dialogs.json", import.meta.url);
  const data = JSON.parse(await readFile(filePath, { encoding: "utf-8" }));
  return data.dialogs || data;
}

export async function createDialog(payload) {
  const newDialog = {
    id: `dlg-${Date.now().toString().slice(-4)}`,
    title: payload?.title || "Новый диалог",
    topic: payload?.topic || "chat",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  };
  return newDialog;
}

export async function getUser() {
  const filePath = new URL("./user.json", import.meta.url);
  const data = JSON.parse(await readFile(filePath, { encoding: "utf-8" }));
  return data.user || data;
}

export async function getTickets() {
  const filePath = new URL("./tickets.json", import.meta.url);
  const data = JSON.parse(await readFile(filePath, { encoding: "utf-8" }));
  return data.tickets || data;
}
