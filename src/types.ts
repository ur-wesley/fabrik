export interface ExtensionContext {
  ui?: {
    notify?(message: string, level?: 'info' | 'warn' | 'error' | 'success'): void;
  };
  model?: {
    set?(modelName: string): void;
  };
}

export interface ExtensionAPI {
  registerCommand(
    name: string,
    definition: {
      description: string;
      handler: (args: string, ctx: ExtensionContext) => Promise<void> | void;
    },
  ): void;

  registerTool(tool: {
    name: string;
    description: string;
    execute: (args: Record<string, unknown>, ctx: ExtensionContext) => Promise<unknown>;
  }): void;

  on?(event: string, handler: (payload: Record<string, unknown>) => Promise<void> | void): void;
}
