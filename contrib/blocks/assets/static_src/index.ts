import { BlockApp } from "./app";
import { SortableController } from "./controllers";
import { BlockController } from "./blocks/controller";
import { ListBlock } from "./blocks/impl/list-block";
import { FieldBlock } from "./blocks/impl/field-block";
import { StructBlock } from "./blocks/impl/struct-block";
import { BoundWidget, CheckboxInput, FileWidget, RadioSelect, Select, Widget } from "./widgets/widget";
import { BoundStreamBlock, BoundStreamBlockValue, StreamBlock } from "./blocks/impl/stream-block";
import { BoundSequenceBlock } from "./blocks/impl/sequence-block";
import { Block, BoundBlock } from "./blocks/base";

window.blocks = new BlockApp();
window.Block = Block;
window.BoundBlock = BoundBlock;
window.BoundSequenceBlock = BoundSequenceBlock;
window.StreamBlock = StreamBlock;
window.BoundStreamBlock = BoundStreamBlock;
window.BoundStreamBlockValue = BoundStreamBlockValue;
window.BoundWidget = BoundWidget;

window.telepath.register('django.widgets.Widget', Widget);
window.telepath.register('django.widgets.SelectWidget', Select);
window.telepath.register('django.widgets.FileWidget', FileWidget);
window.telepath.register('django.widgets.RadioSelect', RadioSelect);
window.telepath.register('django.widgets.CheckboxInput', CheckboxInput);

window.telepath.register('django.blocks.FieldBlock', FieldBlock);
window.blocks.registerBlock('django.blocks.field-block', FieldBlock);

window.telepath.register('django.blocks.ListBlock', ListBlock);
window.blocks.registerBlock('django.blocks.list-block', ListBlock);

window.telepath.register('django.blocks.StructBlock', StructBlock);
window.blocks.registerBlock('django.blocks.struct-block', StructBlock);

window.telepath.register('django.blocks.StreamBlock', StreamBlock);
window.blocks.registerBlock('django.blocks.stream-block', StreamBlock);

window.AdminSite.registerController(
    "sortable", 
    SortableController,
);
window.AdminSite.registerController(
    "block",
    BlockController,
);