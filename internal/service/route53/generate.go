//go:generate go run -tags generate ../../generate/tags/main.go -ListTags=yes -ListTagsInIDElem=ResourceId -ListTagsOutTagsElem=ResourceTagSet.Tags -ServiceTagsSlice=yes -TagOp=ChangeTagsForResource -TagInIDElem=ResourceId -TagInTagsElem=AddTags -TagResTypeElem=ResourceType -UntagOp=ChangeTagsForResource -UntagInTagsElem=RemoveTagKeys -UpdateTags=yes
// ONLY generate directives and package declaration! Do not add anything else to this file.

package route53