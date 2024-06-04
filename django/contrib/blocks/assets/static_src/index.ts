import { SortableController } from "./controllers";
import { BlockController } from "./blocks/controller";
import { Application } from '@hotwired/stimulus';
import type { Definition } from '@hotwired/stimulus';
import { DjangoApplication } from "./app/app";
import { ListBlockDef } from "./blocks/impl/list-block";
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

window.Django.registerBlock('Django.blocks.list-block', ListBlockDef);
window.Django.registerBlock('Django.blocks.field-block', FieldBlockDef);
window.Django.registerBlock('Django.blocks.struct-block', StructBlockDef);

for (let i = 0; i < controllerDefinitions.length; i++) {
    const { identifier, controllerConstructor } = controllerDefinitions[i];
    app.registerController(identifier, controllerConstructor);
}

app.start();

export {
    app,
    SortableController,
};
