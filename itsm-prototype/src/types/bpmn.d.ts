declare module 'bpmn-js' {
  export interface BPMNViewerOptions {
    container?: string | HTMLElement;
    width?: string | number;
    height?: string | number;
    additionalModules?: any[];
    moddleExtensions?: any;
  }

  export interface BPMNModelerOptions extends BPMNViewerOptions {
    keyboard?: any;
    bpmnRenderer?: any;
    textRenderer?: any;
    paletteProvider?: any;
    contextPadProvider?: any;
    popupMenuProvider?: any;
    directEditingProvider?: any;
    elementFactory?: any;
    elementRegistry?: any;
    modeling?: any;
    propertiesProvider?: any;
  }

  export interface BPMNElement {
    id: string;
    type: string;
    businessObject?: any;
    parent?: any;
    children?: any[];
    incoming?: any[];
    outgoing?: any[];
  }

  export interface BPMNShape extends BPMNElement {
    x?: number;
    y?: number;
    width?: number;
    height?: number;
  }

  export interface BPMNConnection extends BPMNElement {
    source?: BPMNElement;
    target?: BPMNElement;
    waypoints?: Array<{ x: number; y: number }>;
  }

  export interface BPMNModdle {
    create(type: string, attrs?: any): any;
    fromXML(xml: string, options?: any): Promise<any>;
    toXML(element: any, options?: any): Promise<string>;
  }

  export interface BPMNModeling {
    createShape(attrs: any, parent?: any, parentIndex?: number): BPMNShape;
    createConnection(attrs: any, source: BPMNElement, target: BPMNElement, parent?: any): BPMNConnection;
    removeElements(elements: BPMNElement[]): void;
    moveElements(elements: BPMNElement[], delta: { x: number; y: number }, target?: any): void;
    resizeShape(shape: BPMNShape, newBounds: { x: number; y: number; width: number; height: number }): void;
    connect(source: BPMNElement, target: BPMNElement, attrs?: any, parent?: any): BPMNConnection;
    updateProperties(element: BPMNElement, properties: any): void;
  }

  export interface BPMNElementRegistry {
    get(id: string): BPMNElement;
    getAll(): BPMNElement[];
    filter(filter: (element: BPMNElement) => boolean): BPMNElement[];
  }

  export interface BPMNCommandStack {
    execute(command: any): void;
    undo(): void;
    redo(): void;
    canUndo(): boolean;
    canRedo(): boolean;
    clear(): void;
  }

  export interface BPMNCanvas {
    zoom(level: number, center?: { x: number; y: number }): void;
    getZoom(): number;
    scrollToElement(element: BPMNElement, options?: any): void;
    addMarker(element: BPMNElement, marker: string): void;
    removeMarker(element: BPMNElement, marker: string): void;
  }

  export interface BPMNEventBus {
    on(event: string, callback: Function, priority?: number): void;
    off(event: string, callback: Function): void;
    fire(event: string, data?: any): void;
  }

  export class BPMNViewer {
    constructor(options?: BPMNViewerOptions);
    importXML(xml: string, options?: any): Promise<any>;
    saveXML(options?: any): Promise<{ xml: string }>;
    saveSVG(options?: any): Promise<{ svg: string }>;
    get(key: string): any;
    destroy(): void;
    on(event: string, callback: Function): void;
    off(event: string, callback: Function): void;
  }

  export class BPMNModeler extends BPMNViewer {
    constructor(options?: BPMNModelerOptions);
    createDiagram(options?: any): Promise<any>;
    get(key: string): any;
  }

  export interface BPMNJS {
    BPMNViewer: typeof BPMNViewer;
    BPMNModeler: typeof BPMNModeler;
  }
}

declare module '@bpmn-io/properties-panel' {
  export interface PropertiesPanelOptions {
    parent: HTMLElement;
    modeler: any;
  }

  export class PropertiesPanel {
    constructor(options: PropertiesPanelOptions);
    attachTo(parent: HTMLElement): void;
    detach(): void;
  }
}

declare module '@bpmn-io/element-template-chooser' {
  export interface ElementTemplateChooserOptions {
    parent: HTMLElement;
    modeler: any;
  }

  export class ElementTemplateChooser {
    constructor(options: ElementTemplateChooserOptions);
    attachTo(parent: HTMLElement): void;
    detach(): void;
  }
}
