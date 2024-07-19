import os

from diagrams.programming.flowchart import Action

os.environ["PATH"] = ";".join(["C:\\Program Files\\Graphviz\\bin", os.environ["PATH"]])

import sys

from diagrams import Diagram, Cluster, Edge

output_file = sys.argv[1]

graph_attr = {
    "fontsize": "12",
    "color": "oranges9",
    "bgcolor": "grey",
    "layout": "dot",
}

with Diagram("yutc", outformat="png", filename="docs/diagram", show=False, graph_attr=graph_attr,
           ) as D, Cluster("yutc"):
    # command
    yutc = Action("yutc", )

    # sub-commands
    template = Action("template")
    forEach = Action("forEach")

    # functions
    loadTemplates = Action("loadTemplates", args="args")
    executeTemplate = Action("executeTemplate", )

    yutc >> template
    yutc >> forEach

    loadTemplates >> Edge(style="rounded", color="oranges9") >> executeTemplate
