// coupon-generator
// coupon-generator-controls
// coupon-generator-remove
// coupon-generator-remove-all
// coupon-generator-add-all
// coupon-generator-add
// coupon-generator-chooser
// data-deselected
// data-selected

import Sortable from 'sortablejs';
import { Controller } from "@hotwired/stimulus";

type CPT_TYPE = Number;

const CPT_RANDOM:  CPT_TYPE = 0;
const CPT_STATIC:  CPT_TYPE = 1;
const CPT_YEAR:    CPT_TYPE = 2;
const CPT_MONTH:   CPT_TYPE = 3;
const CPT_WEEKDAY: CPT_TYPE = 4;
const CPT_DAY:     CPT_TYPE = 5;
const CPT_HOUR:    CPT_TYPE = 6;
const CPT_MINUTE:  CPT_TYPE = 7;
const CPT_UUID:    CPT_TYPE = 8;
const CPT_CONV: any = {
    CPT_RANDOM:  CPT_RANDOM,
    CPT_STATIC:  CPT_STATIC,
    CPT_YEAR:    CPT_YEAR,
    CPT_MONTH:   CPT_MONTH,
    CPT_WEEKDAY: CPT_WEEKDAY,
    CPT_DAY:     CPT_DAY,
    CPT_HOUR:    CPT_HOUR,
    CPT_MINUTE:  CPT_MINUTE,
    CPT_UUID:    CPT_UUID,
}

type CodePart = {
	// type of the code part
	type: CPT_TYPE;

	// length of the code part
	// is only used in case of random or uuid code parts
	length: Number;

	// value of the code part
	// is only used in case of static code parts
	value: string
}

class CouponGeneratorController extends Controller<any> {
    static values = {
          url: String,
    };
    declare urlValue: string;
    declare hasUrlValue: boolean;
    declare hasControlsTarget: boolean;
    declare hasOutputTarget: boolean;
    declare hasInputTarget: boolean;
    declare hasRefreshTarget: boolean;
    declare inputTarget: HTMLElement;
    declare outputTarget: HTMLElement;
    declare controlsTarget: HTMLElement;
    declare partTargets: HTMLElement[];
    declare refreshTarget: HTMLElement;
    declare sortable: Sortable;
    declare out: Sortable;
    static targets = [
        "input",
        "output",
        "part",
        "controls",
        "refresh",
    ];

    connect() {
        if (!this.hasUrlValue) {
            throw new Error("generator API url is required.")
        }
        if (!this.hasInputTarget) {
            throw new Error("input target is required.")
        }
        if (!this.hasControlsTarget) {
            throw new Error("controls target is required.")
        }
        if (!this.hasRefreshTarget) {
            throw new Error("refresh target is required.")
        }

        this.sortable = new Sortable(this.controlsTarget, {
            group: {
                name: `sortableGroup-${this.outputTarget.id}`,
                pull: 'clone', // To clone: set pull to 'clone'
                put: false,
            },
            draggable: ".coupon-generator-part",
            animation: 150,
            sort: false,
        })

        this.out = new Sortable(this.inputTarget, {
            group: `sortableGroup-${this.outputTarget.id}`,
            draggable: ".coupon-generator-part",
            animation: 150,
            onAdd: this.compileParts.bind(this),
        })

        this.refreshTarget.addEventListener("click", this.compileParts.bind(this))
    }

    compileParts() {
        var l = [];
        var partTargets = this.inputTarget.querySelectorAll(".coupon-generator-part") as NodeListOf<HTMLElement>;
        for (let i = 0; i < partTargets.length; i++) {
            let target = partTargets[i];
            let inputLen = target.querySelector("input[type=number]") as HTMLInputElement;
            let inputVal = target.querySelector("input[type=text]") as HTMLInputElement;
            let part: CodePart = {
                type: CPT_CONV[target.dataset.partValue],
                length: inputLen != undefined ? parseInt(inputLen.value ? inputLen.value : "0") : 0,
                value: inputVal != undefined ? inputVal.value : "",
            }

            l.push(part)
        }

        console.log(l)
    }
}

window.Stimulus.register('coupon-generator', CouponGeneratorController);

export {
    CouponGeneratorController
};
