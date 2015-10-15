function InputTools() {
    this.addClicks();
    this.requiredClicks();
}
InputTools.prototype = {
    inputTypes: ['input', 'textarea', 'select'],
    getParent: function (elem) {
        var found = false;
        var parent = elem.parentNode;
        while (!found) {
            if (parent.id === 'parent') {
                found = true;
            } else {
                parent = parent.parentNode;
            }
        }
        return parent;
    },
    getKeys: function (obj) {
        var keys = [];
        for (var key in obj) {
            keys.push(key);
        }
        return keys;
    },
    arrayHas: function (array, check) {
        return array.indexOf(check) > -1;
    },
    requiredInputIsValid: function (input) {
        var re = /[a-zA-Z0-9].+/;
        if (input.required && !re.test(input.value)) {
            this.displayInvalid(input);
            return false;
        }
        return true;
    },
    getJSON: function (validate) {
        this.removeInvalid();
        var inputs = document.querySelectorAll(this.inputTypes.join(', '));
        var data = [];
        var invalid = false;
        for (var i = 0; i < inputs.length; i++) {
            var elem = inputs[i];
            if (!validate || this.requiredInputIsValid(elem)) {
                var out;
                if (elem.id === 'multi') {
                    out = this.parseMulti(elem);
                    i = i + this.getParent(elem).querySelectorAll(this.inputTypes.join(', ')).length - 1;
                } else {
                    out = this.parseInput(elem);
                }
                if (out !== null && out !== {} && out !== []) {
                    data.push(out);
                }
            } else {
                invalid = true;
            }
        }
        if (invalid) {
            return null;
        }
        return data;
    },
    parseInput: function (elem) {
        var o = {};
        if (elem.type === 'checkbox') {
            if (elem.checked) {
                o[elem.name] = true;
            } else {
                o[elem.name] = false;
            }
        } else if (elem.type === 'radio') {
            if (elem.checked) {
                o[elem.name] = elem.value;
            } else {
                o = null;
            }
        } else {
            o[elem.name] = elem.value;
        }
        return o;
    },
    parseMulti: function (elem) {
        var parent = this.getParent(elem);
        var key = parent.getAttribute('data-group');
        var rows = parent.querySelectorAll('.multiple');
        var table = [];
        var out = {};
        table.push(this.parseHeader(rows[0]));
        for (var i = 0; i < rows.length; i++) {
            table.push(this.parseRow(rows[i]));
        }
        out[key] = table;
        return out;
    },
    parseHeader: function (firstRow) {
        var inputs = firstRow.querySelectorAll(this.inputTypes.join(', '));
        var head = [];
        for (var i = 0; i < inputs.length; i++) {
            if (!this.arrayHas(head, inputs[i].name)) {
                head.push(inputs[i].name);
            }
        }
        return head;
    },
    parseRow: function (row) {
        var inputs = row.querySelectorAll(this.inputTypes.join(', '));
        var vals = [];
        for (var i = 0; i < inputs.length; i++) {
            var input = inputs[i];
            if (input.type === 'checkbox') {
                if (input.checked) {
                    vals.push(true);
                } else {
                    vals.push(false);
                }
            } else if (input.type === 'radio') {
                if (input.checked) {
                    vals.push(input.value);
                }
            } else {
                vals.push(input.value);
            }
        }
        return vals;
    },
    displayInvalid: function (invalidInput) {
        var invalidMessage = document.createElement('span');
        var invalidText = document.createTextNode(' * Invalid Text ');
        invalidMessage.style.color = 'red';
        invalidMessage.appendChild(invalidText);
        invalidMessage.id = 'invalid';
        var parent = invalidInput.parentNode;
        parent.insertBefore(invalidMessage, invalidInput);
    },
    removeInvalid: function () {
        var invalid = document.querySelectorAll('#invalid');
        for (var i = 0; i < invalid.length; i++) {
            invalid[i].remove();
        }
    },
    copy: function (elem, data) {
        var multis = this.getParent(elem).querySelectorAll('.multiple');
        var copiedElem = multis[0].cloneNode(true);
        this.resetInputs(copiedElem);
        this.makeRemoveButton(copiedElem);
        if (data !== null && data !== undefined) {
            this.fillCopyInputs(copiedElem, data);
        }
        multis[multis.length - 1].parentNode.insertBefore(copiedElem, multis[multis.length - 1].nextSibling);
    },
    makeRemoveButton: function (elem) {
        var removeButton = document.createElement('button');
        // add optional styling
        removeButton.className = 'btn btn-danger btn-sm';
        var buttonText = document.createTextNode("Remove");
        removeButton.appendChild(buttonText);
        removeButton.onclick = function () {
            elem.remove();
        };
        var buttonHolder = elem.querySelector('#remove-button');
        buttonHolder.insertBefore(removeButton, buttonHolder.childNodes[0]);
    },
    resetInputs: function (elem) {
        var inputs = elem.querySelectorAll('#multi');
        for (var i = 0; i < inputs.length; i++) {
            if (inputs[i].type === 'checkbox' || inputs[i].type === 'radio') {
                inputs[i].checked = false;
            } else {
                inputs[i].value = '';
            }
        }
    },
    fillCopyInputs: function (elem, data) {
        var inputs = elem.querySelectorAll('#multi');
        for (var i = 0; i < inputs.length; i++) {
            if ((inputs[i].type === 'checkbox' || inputs[i].type === 'radio') && data[i] === true) {
                inputs[i].checked = true;
            } else {
                inputs[i].value = data[i];
            }
        }
    },
    fill: function (data) {
        this.removeInvalid();
        for (var i = 0; i < data.length; i++) {
            var key = this.getKeys(data[i])[0];
            if (Array.isArray(data[i][key])) {
                this.fillMulti(data[i][key], key);
            } else {
                this.fillInput(data[i][key], key);
            }
        }
    },
    fillInput: function (data, key) {
        var input = document.querySelectorAll('[name="' + key + '"]');
        if (input[0].type === 'checkbox' && data === true) {
            input[0].checked = true;
        } else if (input[0].type === 'radio') {
            for (var j = 0; j < input.length; j++) {
                if (input[j].value === data) {
                    input[j].checked = true;
                }
            }
        } else {
            input[0].value = data;
        }
    },
    fillMulti: function (data, key) {
        var header = data[0];
        var rows = data.slice(1, data.length);
        for (var i = 0; i < rows.length; i++) {
            var row = rows[i];
            if (i === 0) {
                for (var j = 0; j < row.length; j++) {
                    var val = row[j];
                    var input = document.querySelector('[name="' + header[j] + '"]');
                    if ((input.type === 'checkbox' || input.type === 'radio') && val === true) {
                        input.checked = true;
                    } else {
                        input.value = val;
                    }

                }
            } else {
                this.copy(document.querySelectorAll('[data-group="' + key + '"] .multiple')[0], row);
            }
        }
    },
    validate: function () {
        this.removeInvalid();
        var valid = true;
        var inputs = document.querySelectorAll(this.inputTypes.join(', '));
        for (var i = 0; i < inputs.length; i++) {
            if (!this.requiredInputIsValid(inputs[i])) {
                valid = false;
            }
        }
        return valid;
    },
    getArgs: function () {
        return this.args;
    },
    toggleRequired: function (elem) {
        var inputs = inputTools.getParent(elem).querySelectorAll(this.inputTypes.join(', '));
        for (var i = 0; i < inputs.length; i++) {
            if (elem.checked) {
                inputs[i].required = elem.classList.contains("required");
            } else {
                inputs[i].required = !elem.classList.contains("required");
            }
        }
    },
    requiredClicks: function () {
        var checkboxes = document.querySelectorAll('input[type="checkbox"].required, input[type="checkbox"].invertRequired');
        for (var i = 0; i < checkboxes.length; i++) {
            this.requiredClick(checkboxes[i]);
        }
    },
    requiredClick: function (checkbox) {
        checkbox.onclick = function () {
            inputTools.toggleRequired(this);
        };
    },
    addClicks: function () {
        var addElems = document.querySelectorAll('.add');
        for (var i = 0; i < addElems.length; i++) {
            this.addClick(addElems[i]);
        }
    },
    addClick: function (addElem) {
        addElem.onclick = function () {
            inputTools.copy(this);
        };
    }
};

var inputTools = new InputTools();
