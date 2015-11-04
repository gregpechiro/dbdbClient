function Uploader() {
    this.init()
}

Uploader.prototype = {
    fileTypes: [],
    fileTypeErrorMsg: "",
    defaultText: "",
    maxSize: 0,
    maxSizeMsg: "",
    updateFileInfo: function(e) {
        var t = e.value;
        var n = t.match(/([^\/\\]+)$/);
        var r;
        if (n == null) {
            r = this.defaultText;
        } else {
            r = n[1];
        }
        $('label[for^="' + e.id + '"]').text(r);
        var i = $('form[id="uploader"] input');
        var s = true;
        for (var o = 0; o < i.length; o++) {
            if (i[o].value == "") {
                s = false;
            }
        }
        if (s) {
            $('button.uploader').removeAttr("disabled")
        } else {
            $('button.uploader').attr("disabled", "disabled")
        }
    },
    fileCheck: function(e) {
        if ($('input[id="' + e.id + '"]')[0].files.length > 0) {
            $('div[id="fileError"]').addClass("hide");
            var t = $('input[id="' + e.id + '"]')[0].files[0].size;
            var n = $('input[id="' + e.id + '"]')[0].files[0].type;
            if (t > this.maxSize) {
                $('input[id="' + e.id + '"]')[0].type = "text";
                $('input[id="' + e.id + '"]')[0].type = "file";
                $('p[id="fileMessage"]').html(this.maxSizeMsg);
                $('div[id="fileError"]').removeClass("hide");
                return
            }
            console.log(n);
        	if (this.fileTypes.indexOf(n) > -1) {
        		$('div[id="fileError"]').addClass("hide");
        		return;
        	} else {
        		$('input[id="' + e.id + '"]')[0].type = "text";
        		$('input[id="' + e.id + '"]')[0].type = "file";
        		$('p[id="fileMessage"]').html(this.fileTypeErrorMsg);
        		$('div[id="fileError"]').removeClass("hide");
        	}
        }
    },
    init: function() {
        $('button[id="upload"]').click(function() {
            $('#importModal').modal('hide');
            $('#content').addClass("hide");
            $('div[id="uploadSpinner"]').removeClass("hide");
        });
        $("input.uploader").change(function() {
            uploader.fileCheck(this);
            uploader.updateFileInfo(this);
        })
    }
}

var uploader = new Uploader();
