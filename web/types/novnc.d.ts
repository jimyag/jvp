declare module "@novnc/novnc/lib/rfb" {
  export interface RFBCredentials {
    password?: string;
    username?: string;
    target?: string;
  }

  export interface RFBOptions {
    shared?: boolean;
    credentials?: RFBCredentials;
    repeaterID?: string;
    wsProtocols?: string[];
  }

  export default class RFB extends EventTarget {
    constructor(target: HTMLElement, url: string, options?: RFBOptions);

    // Properties
    viewOnly: boolean;
    focusOnClick: boolean;
    clipViewport: boolean;
    dragViewport: boolean;
    scaleViewport: boolean;
    resizeSession: boolean;
    showDotCursor: boolean;
    background: string;
    qualityLevel: number;
    compressionLevel: number;
    capabilities: { power: boolean };

    // Methods
    disconnect(): void;
    sendCredentials(credentials: RFBCredentials): void;
    sendKey(keysym: number, code: string | null, down?: boolean): void;
    sendCtrlAltDel(): void;
    focus(): void;
    blur(): void;
    machineShutdown(): void;
    machineReboot(): void;
    machineReset(): void;
    clipboardPasteFrom(text: string): void;

    // Events
    addEventListener(type: "connect", listener: (event: Event) => void): void;
    addEventListener(type: "disconnect", listener: (event: CustomEvent<{ clean: boolean }>) => void): void;
    addEventListener(type: "credentialsrequired", listener: (event: Event) => void): void;
    addEventListener(type: "securityfailure", listener: (event: CustomEvent<{ status: number; reason: string }>) => void): void;
    addEventListener(type: "clipboard", listener: (event: CustomEvent<{ text: string }>) => void): void;
    addEventListener(type: "bell", listener: (event: Event) => void): void;
    addEventListener(type: "desktopname", listener: (event: CustomEvent<{ name: string }>) => void): void;
    addEventListener(type: "capabilities", listener: (event: CustomEvent<{ capabilities: { power: boolean } }>) => void): void;
  }
}
