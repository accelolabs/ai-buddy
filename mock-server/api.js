import { readFile } from 'node:fs/promises';

export async function getPatients() {
  const filePath = new URL('./patients.json', import.meta.url);
  const data = JSON.parse(await readFile(filePath, { encoding: 'utf-8' }));
  return data.patients || data;
}

export async function getDialogs() {
  const filePath = new URL('./dialogs.json', import.meta.url);
  const data = JSON.parse(await readFile(filePath, { encoding: 'utf-8' }));
  return data.dialogs || data;
}

export async function getUser() {
  const filePath = new URL('./user.json', import.meta.url);
  const data = JSON.parse(await readFile(filePath, { encoding: 'utf-8' }));
  return data.user || data;
}
