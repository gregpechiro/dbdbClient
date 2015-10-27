$(document).ready(function() {

    Chart.defaults.global.responsive = true;
    //Chart.defaults.global.maintainAspectRatio = false;


    
    var names = [];
    var docCount = [];
    var sizes = [];
    var prettySizes = [];
    var diskLabel = [];
    var deletes = [];
    var adds = [];
    var colors = [];
    var hoverColors = [];

    var totalDel = 0;
    var totalRecords = 0;
    var totalDisk = 0;
    var totalAdds = 0;

    if (stores.length > 0) {
        for (i in stores) {
            names.push(stores[i].Name);
            docCount.push(stores[i].Docs);
            deletes.push(stores[i].Id - stores[i].Docs);
            sizes.push(stores[i].Size);
            prettySizes.push(stores[i].SizePretty);
            diskLabel.push(stores[i].SizePretty);
            adds.push(stores[i].Id);
            colors.push(strToHex(stores[i].Name));
            hoverColors.push(colorChange(colors[i], -0.5));

            totalDel += (stores[i].Id - stores[i].Docs);
            totalRecords += stores[i].Docs;
            totalDisk += stores[i].Size;
            totalAdds += stores[i].Id;
        }
        if (totalDisk >= 1024) {
            totalDisk = decimalRound((totalDisk / 1024), 2) + ' MB';
        } else {
            totalDisk = totalDisk + ' KB';
        }
    }

    $('#totalRecords').html('Total Records: ' + totalRecords);
    $('#totalDisk').html('Total Disk Usage: ' + totalDisk);

    $('#totalDel').html('Total Deletes: ' + totalDel);
    $('#totalAdds').html('Total Adds: ' + totalAdds);
    // $('#totalRead').html('Total Reads: ' + totalDel);
    // $('#totalUpdate').html('Total Updates: ' + totalDel);


    var recordsData = {
        labels: names,
        datasets: [
            {
                label: "Records",
                fillColor: "rgba(231,55,13,1)",
                strokeColor: "rgba(119, 119, 119,1)",
                highlightFill: "rgba(183, 37, 3, 1)",
                highlightStroke: "rgba(67, 67, 67,1)",
                data: docCount
            }
        ]
    }

    var recordsCtx = document.getElementById("recordsChart").getContext("2d");
    var recordsChart = new Chart(recordsCtx).Bar(recordsData);

    var diskData = [];
    for (i in names) {
        var dat = {
            value: sizes[i],
            color: colors[i],
            highlight: hoverColors[i],
            label: names[i]
        }
        diskData.push(dat);
    }
    var diskCtx = document.getElementById("diskChart").getContext("2d");
    var diskChart = new Chart(diskCtx).Pie(diskData, {
        tooltipTemplate: function(valueObject) {
            var val = valueObject.value + ' KB';
            if (valueObject.value > 1024) {
                val = decimalRound((valueObject.value / 1024), 2) + ' MB';
            }
            return valueObject.label + ' : ' + val;
        }
    });

    var deleteData = [];
    for (i in deletes) {
        var dat = {
            value: deletes[i],
            color: colors[i],
            highlight: hoverColors[i],
            label: names[i]
        }
        deleteData.push(dat);
    }
    var deleteCtx = document.getElementById("deleteChart").getContext("2d");
    var deleteChart = new Chart(deleteCtx).PolarArea(deleteData);

    var addData = [];
    for (i in deletes) {
        var dat = {
            value: adds[i],
            color: colors[i],
            highlight: hoverColors[i],
            label: names[i]
        }
        addData.push(dat);
    }
    var addCtx = document.getElementById("addChart").getContext("2d");
    var addChart = new Chart(addCtx).PolarArea(addData);

    /*
    var readData = [];
    for (i in deletes) {
        var dat = {
            value: deletes[i],
            color: colors[i],
            highlight: hoverColors[i],
            label: names[i]
        }
        readData.push(dat);
    }
    var readCtx = document.getElementById("readChart").getContext("2d");
    var readChart = new Chart(readCtx).PolarArea(readData);

    var updateData = [];
    for (i in deletes) {
        var dat = {
            value: deletes[i],
            color: colors[i],
            highlight: hoverColors[i],
            label: names[i]
        }
        updateData.push(dat);
    }
    var updateCtx = document.getElementById("updateChart").getContext("2d");
    var updateChart = new Chart(updateCtx).PolarArea(updateData);
    */

    for (var i = 0; i < names.length; i++) {
        $('span[id="' + names[i] + '"]').css('background-color', colors[i]);
    }
});
