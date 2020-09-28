#!/usr/bin/env python3

from aws_cdk import core

from hitter.hitter_stack import HitterStack

app = core.App()
HitterStack(app, "hitter")

app.synth()
