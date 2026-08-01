import { BlockApp } from "./app";
import {
    Widget,
    BoundWidget,
    InputNotFoundError,
    CheckboxInput,
    RadioSelect,
    Select,
    BoundCheckboxInput,
    BoundRadioSelect,
    BoundSelect,
} from "./widgets/widget";

export {};

declare global {
    interface Window {
        blocks: BlockApp;
        Widget: typeof Widget;
        BoundWidget: typeof BoundWidget;
        InputNotFoundError: typeof InputNotFoundError;
        CheckboxInput: typeof CheckboxInput;
        RadioSelect: typeof RadioSelect;
        Select: typeof Select;
        BoundCheckboxInput: typeof BoundCheckboxInput;
        BoundRadioSelect: typeof BoundRadioSelect;
        BoundSelect: typeof BoundSelect;
    }
}