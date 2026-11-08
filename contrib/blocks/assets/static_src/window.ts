import { BlockApp } from "./app";
import { Block, BoundBlock } from "./blocks/base";
import { BoundSequenceBlock, BoundSequenceBlockValue } from "./blocks/impl/sequence-block";
import { BoundStreamBlock, BoundStreamBlockValue, StreamBlock } from "./blocks/impl/stream-block";
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
    FileWidget,
    BoundFileWidget,
} from "./widgets/widget";

export {};

declare global {
    interface Window {
        blocks: BlockApp;
        Widget: typeof Widget;
        Block: typeof Block;
        BoundBlock: typeof BoundBlock;
        StreamBlock: typeof StreamBlock;
        BoundStreamBlock: typeof BoundStreamBlock;
        BoundStreamBlockValue: typeof BoundStreamBlockValue;
        BoundWidget: typeof BoundWidget;
        BoundSequenceBlock: typeof BoundSequenceBlock;
        BoundSequenceBlockValue: BoundSequenceBlockValue;
        InputNotFoundError: typeof InputNotFoundError;
        CheckboxInput: typeof CheckboxInput;
        RadioSelect: typeof RadioSelect;
        Select: typeof Select;
        FileWidget: typeof FileWidget;
        BoundFileWidget: typeof BoundFileWidget;
        BoundCheckboxInput: typeof BoundCheckboxInput;
        BoundRadioSelect: typeof BoundRadioSelect;
        BoundSelect: typeof BoundSelect;
    }
}