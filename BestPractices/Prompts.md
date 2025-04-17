
## Starting new

When starting a project new we have found it is easier to brainstorm design before writing any code.   This seemed to
have worked for the System Design Platform:

### Prompt 1 - Brainstorming

```
Ok let us brainstorm how this product would look like.  I DO NOT WANT ANY CODE yet.  

For now I am focussed on building a single interface - the "Composer" page.   The main content presentation will be the
form of sections in a blog page.   Since the focus is on system design, the sections should flow like parts of a system
design interview.   I want a seperate TableOfContents component that will allow easy jumping to any section I want.
Sections can have 3 kinds of content: 

* Markdown (with a rich text editor that includes images).
* Drawing (which will be a custom component that you can assume exists). Drawings will be used to draw architecture diagrams. You do not need to build this - it already exists.
* Plots to show system latencies, throughputs, SLOs etc.

Again DO NOT GENERATE code.  I am interested in the interface.   The composer may have a jupyter notebook feel to it so
i can add/delete/move sections.  Sections can be in edit mode which will allow user to update the underlying code/source
and in render mode which will show the rendered version of the section.   

Let us brainstorm
```

### Prompt 2 - Mock Images

After a description and flow were generated, asked it to create mock images ("continue" if over limit):

```
can you generate mock images of this?
```

This gave pretty impressive svg mocks.

### Prompt 3 - Extra requirements to refine

After this was even able to add a requirement ("continue" on outputs):

```
Great I want to add another requirement.  You dont need to change all diagrams.  As a user adds a section, I want a
button that will bring up a dialog titled "Ask the LLM".  This will allow the user to do two things by integrating with
an LLM:

* Ask for a solution for the particular section based on a preset prompt.
* Ask for a solution for the particular section based on their own prompt.
* Ask the LLM to verify the solution based on the other sections and/or their own prompt. For example if the user was
  designing Twitter, the user may ask the LLM to verify if their requirements are enough for the design.
```

### Prompt 4 - Generate overall page

Once the various components of the page were done, it was useful to have it generate overall page:

```
Great.  You created some fantastic mocks.  I want to see a mock of the how the entire page look like as it scrolls. (I
dont need to see the LLM prompt dialog)
```

### Prompt 5 - Starting with Code

Now to the code:

```
Great.  Now let us get started in building this.   Your goal is to only provide this reactive layout. I want the site to be in vanilla html and tailwindcss (and javascript if absolutely necessary) as it will be a static site for fast loading.  The name of the site will be "leetcoach".

FOLLOW THESE INSTRUCTIONS:
* DO NOT use REACT OR ANGULAR
* DO NOT BUILD the entire site at first.
* You will only generate the component I ask you to generate and not others.

First I want you to extract the various components from the layouts and mocks we have built so far and explain to me what you plan to generate. 

DO NOT GENERATE CODE YET.
```

Important to note - making it explain before a generation is very useful

### Prompt 6 - Defining Boundaries

Telling what kind of project output we wanted was very useful:

See INSTRUCTIONS.md for an example

At this point you should see the *reasonably* minimal skeleton you can take anywhere you wanted.

## Checkpointing

Always good to checkpoint:

```
Ok let us checkpoint this. What is our current project folder structure looking like at this point?
```

## Taking existing Project

The problem with the above is at somepoint this project becomes *very* big.   And we either gets slow, or expensive or
hallucinatey!  So we want to use regular checkpointing and restarts.  This is very similar to having a project that we
commit and push after we build and test a feature.

Here the goal is to always start a new project (or a chat), seed it with the context, ask for a feature build, test/fix,
commit and destroy the project.

So the steps here are :

### 1. Create Project

First create the project and upload all the (important) files in your codebase (ie all the component, and template
files).

Then ask it to explain the project so it gets some understanding:

```
What can you tell me about the project with the files I have uploaded?
```

### 2. Feature Ask - Setup

Now upload the screenshot, explain current behavior, the gap and ask it to build X for you, ie you are setting up the LLM to wait for your features before building anything.

```
Great. Currently the root of this renders as follows:
I will add to this one bit at a time. Wait for my instructions. I will upload screenshots of either the entire page or components to verify. How does that sound?
```

### 3. Feature Ask

Make the ask (feel free to add more shots if need be)

```
Let us fix a bug with the shared bit of code that powers "Create First Section" and "Add New Section".   When I click on these buttons for the first time I see the following modal popup correctly.

But when I click on any of those 3 buttons I expect there a new section created with some dummy content but nothing happens.  Why is that?   Dont give me entire code but just show diffs.
```

Voila

