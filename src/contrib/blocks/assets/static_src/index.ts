import { BlockApp } from "./app";
import { SortableController } from "./controllers";
import { BlockController } from "./blocks/controller";
import { ListBlock, ListBlockValue } from "./blocks/impl/list-block";
import { FieldBlock } from "./blocks/impl/field-block";
import { StructBlock } from "./blocks/impl/struct-block";
import { CheckboxInput, RadioSelect, Select, Widget } from "./widgets/widget";

window.blocks = new BlockApp();

window.telepath.register('django.widgets.Widget', Widget);
window.telepath.register('django.widgets.SelectWidget', Select);
window.telepath.register('django.widgets.RadioSelect', RadioSelect);
window.telepath.register('django.widgets.CheckboxInput', CheckboxInput);

window.telepath.register('django.blocks.FieldBlock', FieldBlock);
window.blocks.registerBlock('django.blocks.field-block', FieldBlock);

window.telepath.register('django.blocks.ListBlockValue', ListBlockValue);
window.telepath.register('django.blocks.ListBlock', ListBlock);
window.blocks.registerBlock('django.blocks.list-block', ListBlock);

window.telepath.register('django.blocks.StructBlock', StructBlock);
window.blocks.registerBlock('django.blocks.struct-block', StructBlock);

window.AdminSite.registerController(
    "sortable", 
    SortableController,
);
window.AdminSite.registerController(
    "block",
    BlockController,
);