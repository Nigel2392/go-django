(()=>{"use strict";var e={601:(e,t,n)=>{n.d(t,{A:()=>s});var a=n(982),i=n.n(a),o=n(314),r=n.n(o)()(i());r.push([e.id,".page-link-modal-overlay {\n    position: fixed;\n    top: 0;\n    left: 0;\n    width: 100%;\n    height: 100%;\n    background-color: rgba(0, 0, 0, 0.375);\n    z-index: 1000;\n    display: flex;\n    justify-content: center;\n    align-items: center;\n}\n.page-link-modal-overlay[hidden] {\n    display: none;\n}\n.page-link-modal {\n    /* margin-top: -4rem; */\n    position: relative;\n    background-color: white;\n    border-radius: 4px;\n    box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);\n    padding: 1.5rem;\n    max-width: min(100%, 52em);\n    width: 100%;\n}\n.page-link-modal-controls {\n    position: absolute;\n    top: 0.5rem;\n    right: 0.5rem;\n}\n.page-link-modal-control {\n    background-color: transparent;\n    border: 1px solid rgba(0, 0, 0, 0.05);\n    border-radius: 4px;\n    cursor: pointer;\n    display: inline-block;\n    height: 1.5rem;\n    width: 1.5rem;\n    text-align: center;\n    line-height: 1.5rem;\n}\n.page-link-modal-control + .page-link-modal-control {\n    margin-left: 0.5rem;\n}\n.page-link-modal-close {\n    position: relative;\n    color: #ff0000;\n}\n.page-link-modal-close-x {\n    position: absolute;\n    top: 50%;\n    left: 50%;\n    transform: translate(-50%, -50%);\n}\n.page-link-modal-close:hover {\n    color: #ffffff;\n    background-color: #ff0000;\n}\n.page-link-modal-heading {\n\n}\n.page-link-modal-content {\n\n}\n\n.page-link-modal-parent-page,\n.page-link-modal-page {\n    display: flex;\n    flex-direction: row;\n    align-items: center;\n    padding: 0.5rem 0;\n    border-bottom: 1px solid rgba(0, 0, 0, 0.05);\n}\n.page-link-modal-parent-page-heading,\n.page-link-modal-page-heading {\n    margin: 0;\n    font-size: 1.25rem;\n    font-weight: 500;\n    color: #2e1f5e;\n    cursor: pointer;\n}\n.page-link-modal-parent-page-heading {\n    font-size: 1.5rem;\n}\n.page-link-modal-page-down,\n.page-link-modal-parent-page-pageup {\n    display: flex;\n    cursor: pointer;\n    background-size: 1.5rem;\n    background-repeat: no-repeat;\n    background-position: center;\n}\n.page-link-modal-page-down {\n    margin-left: auto;\n}\n.page-link-modal-page-down svg {\n    color: #2e1f5e;\n    width: 1.5rem;\n    height: 1.5rem;\n}\n.page-link-modal-parent-page-pageup svg {\n    color: #2e1f5e;\n    width: 2.5rem;\n    height: 2.5rem;\n}\n\n.page-link-modal-loader{\n  width: 100%;\n  height: 3px;\n  background-color: #2e1f5e;\n  animation: line 2s infinite alternate;\n}\n@keyframes line{\n  0%{\n      transform: scaleX(0);\n      transform-origin: left;\n  }\n  45%{\n      transform: scaleX(1);\n      transform-origin: left;\n  }\n  50%{\n      transform: scaleX(1);\n      transform-origin: right;\n  }\n  55%{\n      transform: scaleX(1);\n      transform-origin: right;\n  }\n  100%{\n      transform: scaleX(0);\n      transform-origin: right;\n  }\n}\n  ",""]);const s=r},314:e=>{e.exports=function(e){var t=[];return t.toString=function(){return this.map((function(t){var n="",a=void 0!==t[5];return t[4]&&(n+="@supports (".concat(t[4],") {")),t[2]&&(n+="@media ".concat(t[2]," {")),a&&(n+="@layer".concat(t[5].length>0?" ".concat(t[5]):""," {")),n+=e(t),a&&(n+="}"),t[2]&&(n+="}"),t[4]&&(n+="}"),n})).join("")},t.i=function(e,n,a,i,o){"string"==typeof e&&(e=[[null,e,void 0]]);var r={};if(a)for(var s=0;s<this.length;s++){var l=this[s][0];null!=l&&(r[l]=!0)}for(var d=0;d<e.length;d++){var p=[].concat(e[d]);a&&r[p[0]]||(void 0!==o&&(void 0===p[5]||(p[1]="@layer".concat(p[5].length>0?" ".concat(p[5]):""," {").concat(p[1],"}")),p[5]=o),n&&(p[2]?(p[1]="@media ".concat(p[2]," {").concat(p[1],"}"),p[2]=n):p[2]=n),i&&(p[4]?(p[1]="@supports (".concat(p[4],") {").concat(p[1],"}"),p[4]=i):p[4]="".concat(i)),t.push(p))}},t}},982:e=>{e.exports=function(e){return e[1]}},72:e=>{var t=[];function n(e){for(var n=-1,a=0;a<t.length;a++)if(t[a].identifier===e){n=a;break}return n}function a(e,a){for(var o={},r=[],s=0;s<e.length;s++){var l=e[s],d=a.base?l[0]+a.base:l[0],p=o[d]||0,c="".concat(d," ").concat(p);o[d]=p+1;var h=n(c),g={css:l[1],media:l[2],sourceMap:l[3],supports:l[4],layer:l[5]};if(-1!==h)t[h].references++,t[h].updater(g);else{var m=i(g,a);a.byIndex=s,t.splice(s,0,{identifier:c,updater:m,references:1})}r.push(c)}return r}function i(e,t){var n=t.domAPI(t);return n.update(e),function(t){if(t){if(t.css===e.css&&t.media===e.media&&t.sourceMap===e.sourceMap&&t.supports===e.supports&&t.layer===e.layer)return;n.update(e=t)}else n.remove()}}e.exports=function(e,i){var o=a(e=e||[],i=i||{});return function(e){e=e||[];for(var r=0;r<o.length;r++){var s=n(o[r]);t[s].references--}for(var l=a(e,i),d=0;d<o.length;d++){var p=n(o[d]);0===t[p].references&&(t[p].updater(),t.splice(p,1))}o=l}}},659:e=>{var t={};e.exports=function(e,n){var a=function(e){if(void 0===t[e]){var n=document.querySelector(e);if(window.HTMLIFrameElement&&n instanceof window.HTMLIFrameElement)try{n=n.contentDocument.head}catch(e){n=null}t[e]=n}return t[e]}(e);if(!a)throw new Error("Couldn't find a style target. This probably means that the value for the 'insert' parameter is invalid.");a.appendChild(n)}},540:e=>{e.exports=function(e){var t=document.createElement("style");return e.setAttributes(t,e.attributes),e.insert(t,e.options),t}},56:(e,t,n)=>{e.exports=function(e){var t=n.nc;t&&e.setAttribute("nonce",t)}},825:e=>{e.exports=function(e){if("undefined"==typeof document)return{update:function(){},remove:function(){}};var t=e.insertStyleElement(e);return{update:function(n){!function(e,t,n){var a="";n.supports&&(a+="@supports (".concat(n.supports,") {")),n.media&&(a+="@media ".concat(n.media," {"));var i=void 0!==n.layer;i&&(a+="@layer".concat(n.layer.length>0?" ".concat(n.layer):""," {")),a+=n.css,i&&(a+="}"),n.media&&(a+="}"),n.supports&&(a+="}");var o=n.sourceMap;o&&"undefined"!=typeof btoa&&(a+="\n/*# sourceMappingURL=data:application/json;base64,".concat(btoa(unescape(encodeURIComponent(JSON.stringify(o))))," */")),t.styleTagTransform(a,e,t.options)}(t,e,n)},remove:function(){!function(e){if(null===e.parentNode)return!1;e.parentNode.removeChild(e)}(t)}}}},113:e=>{e.exports=function(e,t){if(t.styleSheet)t.styleSheet.cssText=e;else{for(;t.firstChild;)t.removeChild(t.firstChild);t.appendChild(document.createTextNode(e))}}}},t={};function n(a){var i=t[a];if(void 0!==i)return i.exports;var o=t[a]={id:a,exports:{}};return e[a](o,o.exports,n),o.exports}n.n=e=>{var t=e&&e.__esModule?()=>e.default:()=>e;return n.d(t,{a:t}),t},n.d=(e,t)=>{for(var a in t)n.o(t,a)&&!n.o(e,a)&&Object.defineProperty(e,a,{enumerable:!0,get:t[a]})},n.o=(e,t)=>Object.prototype.hasOwnProperty.call(e,t),n.nc=void 0,(()=>{function e(e,t,...n){if("function"==typeof e)return e(null!=t?t:{},n);const a=document.createElement(e);if(t)for(const[e,n]of Object.entries(t))e in a?a[e]=n:a.setAttribute(e,n);const i=e=>{"string"==typeof e?a.appendChild(document.createTextNode(e)):e instanceof Node?a.appendChild(e):Array.isArray(e)?e.forEach(i):console.warn("Invalid child type:",e)};return n.forEach(i),a}var t=function(e,t,n,a){return new(n||(n=Promise))((function(i,o){function r(e){try{l(a.next(e))}catch(e){o(e)}}function s(e){try{l(a.throw(e))}catch(e){o(e)}}function l(e){var t;e.done?i(e.value):(t=e.value,t instanceof n?t:new n((function(e){e(t)}))).then(r,s)}l((a=a.apply(e,t||[])).next())}))};class a{constructor(e){this.elements={overlay:null,modal:null,loader:null,close:null,content:null,error:null},this.opts=e}initChooser(){this.elements.overlay=e("div",{class:"page-link-modal-overlay"},e("div",{class:"page-link-modal"},e("div",{class:"page-link-modal-controls"},e("button",{class:"page-link-modal-control page-link-modal-close",type:"button"},e("span",{className:"page-link-modal-close-x"},"×"))),e("div",{class:"page-link-modal-heading"},e("h2",null,this.opts.translate("Choose a Page"))),e("div",{class:"page-link-modal-loader",role:"status",style:"margin-bottom:2%;"}),e("div",{class:"page-link-modal-error"}),e("div",{class:"page-link-modal-content"}))),this.elements.modal=this.elements.overlay.querySelector(".page-link-modal"),this.elements.loader=this.elements.modal.querySelector(".page-link-modal-loader"),this.elements.close=this.elements.modal.querySelector(".page-link-modal-close"),this.elements.content=this.elements.modal.querySelector(".page-link-modal-content"),this.elements.error=this.elements.modal.querySelector(".page-link-modal-error"),this.elements.close.addEventListener("click",(()=>{this.elements.overlay.remove()})),this.opts.openedByDefault&&(this.elements.overlay.hidden=!1,this.opts.modalOpen&&this.opts.modalOpen(this)),document.body.appendChild(this.elements.overlay),this.modalError(null),this.loadPageList()}_try(e,t,...n){try{return t(...n)}catch(t){this.modalError(e)}}loadPages(){return t(this,arguments,void 0,(function*(e=null,t=!1){null==e&&(e=this.state?this.state.id:""),this.modalError(null),this.elements.loader.hidden=!1;const n=new URLSearchParams({[this.opts.pageListQueryVar]:e.toString(),get_parent:t.toString()}),a=this.opts.pageListURL+"?"+n.toString(),i=yield this._try("Error fetching menu items",fetch,a);if(!i.ok)return void this.modalError(this.opts.translate("Error fetching menu items"));const o=yield this._try("Error parsing response",i.json.bind(i));return this.elements.loader.hidden=!0,o}))}loadPageList(){return t(this,arguments,void 0,(function*(t=null,n=!1){this.modalError(null),this.elements.content.innerHTML="",this.elements.loader.hidden=!1;let a=yield this.loadPages(t,n);if(a)if(0!=a.items.length||a.parent_item){if(a.parent_item){const t=e("div",{class:"page-link-modal-parent-page-pageup"});t.innerHTML='<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-arrow-return-left" viewBox="0 0 16 16">\n    <path fill-rule="evenodd" d="M14.5 1.5a.5.5 0 0 1 .5.5v4.8a2.5 2.5 0 0 1-2.5 2.5H2.707l3.347 3.346a.5.5 0 0 1-.708.708l-4.2-4.2a.5.5 0 0 1 0-.708l4-4a.5.5 0 1 1 .708.708L2.707 8.3H12.5A1.5 1.5 0 0 0 14 6.8V2a.5.5 0 0 1 .5-.5"/>\n</svg>';const n=e("div",{class:"page-link-modal-parent-page","data-page-id":a.parent_item.id,"data-depth":a.parent_item.depth},t,e("div",{class:"page-link-modal-parent-page-heading"},a.parent_item.title));t.addEventListener("click",(()=>{this.loadPageList(a.parent_item.id,!0),this._state=a.parent_item})),this.elements.content.appendChild(n),0==a.items.length&&this.elements.content.appendChild(e("div",{class:"page-link-modal-page"},e("div",{class:"page-link-modal-page-heading"},this.opts.translate("No live child pages found"))))}for(let t of a.items){const n=e("div",{class:"page-link-modal-page","data-page-id":t.id,"data-depth":t.depth},e("div",{className:"page-link-modal-page-heading"},t.title));if(t.numchild>0){let a=e("div",{class:"page-link-modal-page-down"});a.innerHTML='<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-box-arrow-in-right" viewBox="0 0 16 16">\n    <path fill-rule="evenodd" d="M6 3.5a.5.5 0 0 1 .5-.5h8a.5.5 0 0 1 .5.5v9a.5.5 0 0 1-.5.5h-8a.5.5 0 0 1-.5-.5v-2a.5.5 0 0 0-1 0v2A1.5 1.5 0 0 0 6.5 14h8a1.5 1.5 0 0 0 1.5-1.5v-9A1.5 1.5 0 0 0 14.5 2h-8A1.5 1.5 0 0 0 5 3.5v2a.5.5 0 0 0 1 0z"/>\n    <path fill-rule="evenodd" d="M11.854 8.354a.5.5 0 0 0 0-.708l-3-3a.5.5 0 1 0-.708.708L10.293 7.5H1.5a.5.5 0 0 0 0 1h8.793l-2.147 2.146a.5.5 0 0 0 .708.708z"/>\n</svg>',a.addEventListener("click",(()=>{this.loadPageList(t.id),this._state=t})),n.appendChild(a)}n.querySelector(".page-link-modal-page-heading").addEventListener("click",(()=>{this._state=t,this.opts.pageChosen&&this.opts.pageChosen(t),this.elements.overlay.animate([{opacity:1},{opacity:0}],{duration:200,fill:"forwards"}).onfinish=()=>{this.elements.overlay.hidden=!0}})),this.elements.content.appendChild(n)}}else this.modalError(this.opts.translate("No pages found"))}))}get state(){return this._state}set onOpen(e){this.opts.modalOpen=e}set onClose(e){this.opts.modalClose=e}set onDelete(e){this.opts.modalDelete=e}set onChosen(e){this.opts.pageChosen=e}open(){this.elements.overlay?this.elements.overlay.hidden=!1:this.initChooser(),this.opts.modalOpen&&this.opts.modalOpen(this)}close(){this.elements.overlay.hidden=!0,this.opts.modalClose&&this.opts.modalClose(this)}delete(){this.elements.overlay&&this.elements.overlay.remove(),this.opts.modalDelete&&this.opts.modalDelete(this)}modalError(e){this.elements.loader.hidden=!0,""!=e&&null!=e||(this.elements.error.innerHTML=""),this.elements.error.innerText=e}}var i=n(72),o=n.n(i),r=n(825),s=n.n(r),l=n(659),d=n.n(l),p=n(56),c=n.n(p),h=n(540),g=n.n(h),m=n(113),u=n.n(m),f=n(601),v={};v.styleTagTransform=u(),v.setAttributes=c(),v.insert=d().bind(null,"head"),v.domAPI=s(),v.insertStyleElement=g(),o()(f.A,v),f.A&&f.A.locals&&f.A.locals,window.PageLinkTool=class{constructor({data:e,api:t,config:n}){if(n||(n={}),this.tag="A",this.tagClass="page-link",this.data=e,this.api=t,this.config=n,!this.config.pageListURL||!this.config.retrievePageURL||!this.config.pageListQueryVar)throw new Error("PageLinkTool requires pageMenuURL, retrievePageURL, and pageIDVar in config")}static get isInline(){return!0}static get sanitize(){return{a:!0}}set state(e){this._state=e,this.button.classList.toggle(this.api.styles.inlineToolButtonActive,e)}get state(){return this._state}validate(e){return!!(e&&e.id&&e.text)&&""!==e.id.trim()&&""!==e.text.trim()}surround(e){this.state?this.unwrap(e):(this.modal.open(),this.modal.onChosen=t=>{this.wrap(e,t)})}wrap(e,t){let n=e.extractContents();const a=this.api.selection.findParentTag(this.tag);(a||a&&a.querySelector(this.tag.toLowerCase()))&&a.remove(),this.wrapperTag=document.createElement(this.tag),this.wrapperTag.dataset.pageId=t.id,this.wrapperTag.classList.add(this.tagClass),this.wrapperTag.appendChild(n),e.insertNode(this.wrapperTag),this.api.selection.expandToTag(this.wrapperTag)}unwrap(e){const t=this.api.selection.findParentTag(this.tag),n=e.extractContents();t.remove(),e.insertNode(n)}render(){return this.button=document.createElement("button"),this.button.type="button",this.button.classList.add(this.api.styles.inlineToolButton),this.button.innerHTML='<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-link-45deg" viewBox="0 0 16 16">\n  <path d="M4.715 6.542 3.343 7.914a3 3 0 1 0 4.243 4.243l1.828-1.829A3 3 0 0 0 8.586 5.5L8 6.086a1 1 0 0 0-.154.199 2 2 0 0 1 .861 3.337L6.88 11.45a2 2 0 1 1-2.83-2.83l.793-.792a4 4 0 0 1-.128-1.287z"/>\n  <path d="M6.586 4.672A3 3 0 0 0 7.414 9.5l.775-.776a2 2 0 0 1-.896-3.346L9.12 3.55a2 2 0 1 1 2.83 2.83l-.793.792c.112.42.155.855.128 1.287l1.372-1.372a3 3 0 1 0-4.243-4.243z"/>\n</svg>',this.modal=new a({pageListURL:this.config.pageListURL,pageListQueryVar:this.config.pageListQueryVar,translate:this.api.i18n.t.bind(this.api.i18n)}),this.button}checkState(){const e=this.api.selection.findParentTag(this.tag,this.tagClass);this.state=!!e}save(e){const t=e.querySelector("a");return{id:t.dataset.id,text:t.innerText}}}})()})();