import * as vscode from "vscode";
import { citationToFileLocation } from "../citations";

export function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

export function createNonce(): string {
  return Math.random().toString(36).slice(2) + Math.random().toString(36).slice(2);
}

export async function openWorkspaceLocation(
  workspaceFolder: vscode.WorkspaceFolder,
  filePath: string,
  line?: number
): Promise<void> {
  const target = vscode.Uri.file(`${workspaceFolder.uri.fsPath}/${filePath}`);
  const document = await vscode.workspace.openTextDocument(target);
  const editor = await vscode.window.showTextDocument(document, vscode.ViewColumn.One);
  if (line && line > 0) {
    const position = new vscode.Position(line - 1, 0);
    editor.selection = new vscode.Selection(position, position);
    editor.revealRange(new vscode.Range(position, position));
  }
}

/**
 * Parse a citation handle or legacy "path:line" reference into a file
 * location. Understands the full canonical format:
 *   - "path:startLine-endLine"  → jumps to startLine
 *   - "path:line"               → jumps to that line (legacy single-line)
 *   - "path"                    → opens at the top of the file
 *
 * Delegates to the shared citation parser so QA, compliance, and
 * knowledge artifact handles all parse correctly.
 */
export function parseFileReference(ref: string): { filePath: string; line?: number } {
  // Try the canonical citation parser first — handles path:start-end and sym_ prefixes.
  const loc = citationToFileLocation(ref);
  if (loc) {
    return { filePath: loc.filePath, line: loc.line };
  }
  // Fall through: treat as a bare path (no line info, not a recognized citation).
  return { filePath: ref.trim() };
}
