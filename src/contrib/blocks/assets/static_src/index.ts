import type { Definition } from '@hotwired/stimulus';
import { DjangoApplication } from "./app/app";
import { SortableController } from "./controllers";
import { BlockController } from "./blocks/controller";
import { ListBlockDef, ListBlockValue } from "./blocks/impl/list-block";
import { FieldBlockDef } from "./blocks/impl/field-block";
import { StructBlockDef } from "./blocks/impl/struct-block";

const controllerDefinitions: Definition[] = [
    { identifier: 'sortable', controllerConstructor: SortableController},
    { identifier: 'block', controllerConstructor: BlockController},
];

const app = new DjangoApplication({
    controllers: {},
    blocks: {},
})


window.Django = app;

window.Django.registerAdapter('django.blocks.FieldBlock', FieldBlockDef);
window.Django.registerBlock('django.blocks.field-block', FieldBlockDef);

window.Django.registerAdapter('django.blocks.ListBlockValue', ListBlockValue);
window.Django.registerAdapter('django.blocks.ListBlock', ListBlockDef);
window.Django.registerBlock('django.blocks.list-block', ListBlockDef);

window.Django.registerAdapter('django.blocks.StructBlock', StructBlockDef);
window.Django.registerBlock('django.blocks.struct-block', StructBlockDef);

for (let i = 0; i < controllerDefinitions.length; i++) {
    const { identifier, controllerConstructor } = controllerDefinitions[i];
    app.registerController(identifier, controllerConstructor);
}

app.start();

export {
    app,
    SortableController,
};
