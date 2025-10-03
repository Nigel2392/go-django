import { BlockApp } from "./app";
import { SortableController } from "./controllers";
import { BlockController } from "./blocks/controller";
import { ListBlockDef, ListBlockValue } from "./blocks/impl/list-block";
import { FieldBlockDef } from "./blocks/impl/field-block";
import { StructBlockDef } from "./blocks/impl/struct-block";

window.telepath.register('django.blocks.FieldBlock', FieldBlockDef);
window.blocks.registerBlock('django.blocks.field-block', FieldBlockDef);

window.telepath.register('django.blocks.ListBlockValue', ListBlockValue);
window.telepath.register('django.blocks.ListBlock', ListBlockDef);
window.blocks.registerBlock('django.blocks.list-block', ListBlockDef);

window.telepath.register('django.blocks.StructBlock', StructBlockDef);
window.blocks.registerBlock('django.blocks.struct-block', StructBlockDef);

window.AdminSite.registerController(
    "sortable", 
    SortableController,
);
window.AdminSite.registerController(
    "block",
    BlockController,
);


export {
    BlockApp,
    SortableController,
};
